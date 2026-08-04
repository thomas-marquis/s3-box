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

type csvColumn struct {
	Width float32
}

type csvEditor struct {
	editor.Base

	cancelFunc  func()
	contentHash string

	Records binding.List[[]string]
	Columns binding.List[csvColumn]
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	ed := &csvEditor{
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
		Columns: binding.NewList[csvColumn](func(c1, c2 csvColumn) bool {
			return c1 == c2
		}),
	}

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

func (e *csvEditor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *csvEditor) Save() {
	e.IsLoading.Set(true)          //nolint:errcheck
	e.StatusLabel.Set("Saving...") // nolint:errcheck

	content := e.getContent()
	ctx, cancel := context.WithCancel(context.Background())

	e.Lock()
	e.cancelFunc = cancel
	e.Unlock()

	handleFailure := func(err error) {
		e.StatusLabel.Set("error (unsaved)") //nolint:errcheck
		e.Err.Set(err)                       //nolint:errcheck

		e.Lock()
		//e.shouldCloseWhenSaved = false
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
		e.Lock()
		//if e.shouldCloseWhenSaved {
		//	e.RequestClose()
		//}
		e.Unlock()
	}()
}

func (e *csvEditor) RequestClose() {
	e.Bus.Publish(event.New(editor.CloseRequested{
		Editor: e,
	}))
}

func (e *csvEditor) Cancel() {
	e.Lock()
	defer e.Unlock()

	if e.cancelFunc == nil {
		return
	}
	e.cancelFunc()
	e.cancelFunc = nil
}

func (e *csvEditor) HasChanged() bool {
	e.Lock()
	defer e.Unlock()
	return e.contentHash != sha256Hex(e.getContent())
}

func (e *csvEditor) getContent() string {
	if e.Records.Length() == 0 {
		return ""
	}

	records, _ := e.Records.Get() //nolint:errcheck
	builder := strings.Builder{}
	for _, row := range records {
		for _, cell := range row {
			builder.WriteString(cell)
			builder.WriteString(sep)
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func (e *csvEditor) updateContentHash(newContent string) {
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
