package csveditor

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
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
	shortcutQuit = desktop.CustomShortcut{
		KeyName:  fyne.KeyQ,
		Modifier: fyne.KeyModifierControl,
	}
)

type csvColumn struct {
	Width float32
}

type csvEditor struct {
	editor.Base

	bus         event.Bus
	mu          sync.Mutex
	cancelFunc  func()
	contentHash string

	Records      binding.List[[]string]
	Columns      binding.List[csvColumn]
	IsLoading    binding.Bool
	StatusLabel  binding.String
	ConfirmClose func(onConfirm func(confirmed bool))
	closer       io.Closer
}

var (
	_ editor.Closable = (*csvEditor)(nil)
)

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	ed := &csvEditor{
		Base: editor.NewBase(w, file),
		bus:  bus,
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
		IsLoading:    binding.NewBool(),
		StatusLabel:  binding.NewString(),
		ConfirmClose: func(onConfirm func(confirmed bool)) {},
	}

	ed.IsLoading.Set(true) //nolint:errcheck

	w.Canvas().AddShortcut(&shortcutQuit, func(fyne.Shortcut) {
		ed.closer.Close() //nolint:errcheck
	})
	w.Canvas().AddShortcut(&shortcutSave, func(fyne.Shortcut) {
		ed.Save()
	})

	return ed
}

func (e *csvEditor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *csvEditor) OnLoaded(fileContent directory.FileContent, err error) {
	defer e.IsLoading.Set(false) //nolint:errcheck
	if err != nil {
		e.StatusLabel.Set("error (unloaded)") //nolint:errcheck
		return
	}

	r := csv.NewReader(fileContent)

	nbRows := 0
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		e.Records.Append(record) //nolint:errcheck
		nbRows++
	}

	if e.Records.Length() == 0 {
		return
	}

	e.updateContentHash(e.getContent())

	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)

	firstRow, _ := e.Records.GetValue(0)
	nbCols := len(firstRow)
	for i := range nbCols {
		col := csvColumn{}
		for j := range nbRows {
			row, _ := e.Records.GetValue(j)
			cw := colWidth(row[i], textSize)
			if col.Width < cw-cellPadding {
				col.Width = cw
			}
		}
		e.Columns.Append(col) //nolint:errcheck
	}
}

func (e *csvEditor) Save() {
	e.IsLoading.Set(true)          //nolint:errcheck
	e.StatusLabel.Set("Saving...") //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.cancelFunc = cancel
	e.mu.Unlock()

	e.bus.Publish(event.New(editor.SaveTriggered{
		File:    e.File(),
		Content: e.getContent(),
	}, event.WithContext(ctx)))
}

func (e *csvEditor) OnSaved(newContent string, err error) {
	e.IsLoading.Set(false) //nolint:errcheck

	if err != nil {
		e.StatusLabel.Set("error (unsaved)") //nolint:errcheck
		return
	}

	e.updateContentHash(newContent)
	e.StatusLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05"))) // nolint:errcheck
}

func (e *csvEditor) BeforeClose(cb func(ready bool)) {
	if e.HasChanged() {
		e.ConfirmClose(func(confirmed bool) {
			e.Cancel()
			cb(confirmed)
		})
		return
	}

	e.Cancel()
	cb(true)
}

func (e *csvEditor) SetCloser(closer io.Closer) {
	e.closer = closer
}

// Close triggers a close from within the editor
func (e *csvEditor) Close() {
	e.BeforeClose(func(ready bool) {
		if ready {
			if err := e.closer.Close(); err != nil {
				e.StatusLabel.Set("error (unclosed)") //nolint:errcheck
			}
		}
	})
}

func (e *csvEditor) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancelFunc == nil {
		return
	}
	e.cancelFunc()
	e.cancelFunc = nil
}

func (e *csvEditor) HasChanged() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
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
	e.mu.Lock()
	defer e.mu.Unlock()
	e.contentHash = sha256Hex(newContent)
}

func colWidth(text string, textSize float32) float32 {
	return fyne.MeasureText(text, textSize, fyne.TextStyle{}).Width + cellPadding
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
