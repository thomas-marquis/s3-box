package csveditor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	sep = ","
)

var (
	shortcutSave = desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierControl,
	}
)

type CsvColumn struct {
	Width float32
}

type Editor struct {
	*editor.Base

	cancelFunc  func()
	contentHash string

	Records   binding.List[[]string]
	Columns   binding.List[CsvColumn]
	Paginator *Paginator
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	ed := &Editor{
		Base: editor.NewBase(bus, w, file),
		Records: binding.NewList[[]string](func(l1, l2 []string) bool {
			if len(l1) != len(l2) {
				return false
			}
			for i := range l1 {
				if l1[i] != l2[i] {
					return false
				}
			}
			return true
		}),
		Columns: binding.NewList[CsvColumn](func(c1, c2 CsvColumn) bool {
			return c1 == c2
		}),
	}
	ed.Paginator = NewCsvPaginator(ed.Records)

	ed.ExtendBaseEditor(ed)

	ed.IsLoading.Set(true) //nolint:errcheck

	w.Canvas().AddShortcut(&shortcutSave, func(fyne.Shortcut) {
		ed.Save()
	})

	ed.Sub.
		On(event.Is(editor.LoadedType), ed.handleLoaded).
		On(event.Is(editor.LoadFailedType), ed.handleLoadFailed).
		On(event.Is(editor.CloseRequestedType), ed.handleCloseRequested)
	ed.Sub.ListenWithWorkers(2)

	return ed
}

func (e *Editor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *Editor) NextPage() {
	e.Paginator.Next()
}

func (e *Editor) PrevPage() {
	e.Paginator.Prev()
}

func (e *Editor) Save() {
	e.IsLoading.Set(true)          //nolint:errcheck
	e.StatusLabel.Set("Saving...") // nolint:errcheck

	content := e.GetContent()
	ctx, cancel := context.WithCancel(context.Background())

	e.Lock()
	e.cancelFunc = cancel
	e.Unlock()

	handleFailure := func(err error) {
		e.StatusLabel.Set("error (unsaved)") //nolint:errcheck
		e.Err.Set(err)                       //nolint:errcheck

		e.Lock()
		e.cancelFunc = nil
		e.Unlock()
	}

	if !e.IsLoaded() {
		handleFailure(errors.New("file not loaded"))
		return
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			e.Content.Cancel()
			close(done)
		case <-done:
		}
	}()

	go func() {
		defer close(done)
		defer e.IsLoading.Set(false) //nolint:errcheck

		if _, err := e.Content.Seek(0, io.SeekStart); err != nil {
			handleFailure(err)
			return
		}
		if _, err := fmt.Fprint(e.Content, content); err != nil {
			handleFailure(err)
			return
		}
		e.File().SetSizeBytes(uint64(len(content)))

		e.updateContentHash(content)
		e.StatusLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05"))) // nolint:errcheck
	}()
}

func (e *Editor) RequestClose() {
	e.Bus.Publish(event.New(editor.CloseRequested{
		Editor: e,
	}))
}

func (e *Editor) Cancel() {
	e.Lock()
	defer e.Unlock()

	if e.cancelFunc == nil {
		return
	}
	e.cancelFunc()
	e.cancelFunc = nil
}

func (e *Editor) HasChanged() bool {
	e.Lock()
	defer e.Unlock()
	return e.contentHash != sha256Hex(e.GetContent())
}

func (e *Editor) GetContent() string {
	if len(e.Paginator.Records) == 0 {
		return ""
	}

	builder := strings.Builder{}
	for i, row := range e.Paginator.Records {
		for j, cell := range row {
			builder.WriteString(cell)
			if j < len(row)-1 {
				builder.WriteString(sep)
			}
		}
		if i < len(e.Paginator.Records)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func (e *Editor) updateContentHash(newContent string) {
	e.Lock()
	defer e.Unlock()
	e.contentHash = sha256Hex(newContent)
}

func colWidth(text string, textSize float32) float32 {
	return fyne.MeasureText(text, textSize, fyne.TextStyle{}).Width + cellPadding
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
