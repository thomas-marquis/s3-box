package widget

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
)

type textEditorEntry struct {
	widget.Entry

	onValidate func(string)
}

func (m *textEditorEntry) TypedShortcut(s fyne.Shortcut) {
	if val, ok := s.(*desktop.CustomShortcut); ok {
		if val.KeyName == fyne.KeyS && val.Modifier == fyne.KeyModifierControl {
			m.onValidate(m.Text)
		}
	} else {
		m.Entry.TypedShortcut(s)
	}
}

func newTextEditorEntry(onValidate func(string)) *textEditorEntry {
	e := &textEditorEntry{
		Entry: widget.Entry{
			MultiLine: true,
			Wrapping:  fyne.TextWrap(fyne.TextTruncateClip),
		},
		onValidate: onValidate,
	}
	e.ExtendBaseWidget(e)
	return e
}

type FileEditor struct {
	widget.BaseWidget

	openedEditor *viewmodel.OpenedEditor
	contentHash  string
	stateLabel   binding.String
}

func NewFileEditor(openedEditor *viewmodel.OpenedEditor) *FileEditor {
	w := &FileEditor{
		openedEditor: openedEditor,
		stateLabel:   binding.NewString(),
	}
	w.ExtendBaseWidget(w)

	once := sync.Once{}

	openedEditor.Content.AddListener(binding.NewDataListener(func() {
		once.Do(func() {
			val, _ := openedEditor.Content.Get()
			w.contentHash = sha256Hex(val)
		})
	}))

	openedEditor.Window.SetCloseIntercept(func() {
		val, _ := openedEditor.Content.Get()
		if w.hasChanged(val) {
			dialog.ShowConfirm("Discard changes?", "Do you want to discard your changes?", func(confirmed bool) {
				if confirmed {
					openedEditor.Window.Close()
				}
			}, openedEditor.Window)
		} else {
			openedEditor.Window.Close()
		}
	})

	openedEditor.ErrorMsg.AddListener(binding.NewDataListener(func() {
		msg, _ := openedEditor.ErrorMsg.Get()
		if msg == "" {
			return
		}
		dialog.ShowError(errors.New(msg), openedEditor.Window)
	}))

	return w
}

func (w *FileEditor) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	editor := newTextEditorEntry(w.handleSave)
	editor.Bind(w.openedEditor.Content)

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			w.handleSave(editor.Text)
		}),
	)

	loader := widget.NewProgressBarInfinite()
	loader.Hide()
	w.openedEditor.IsLoaded.AddListener(binding.NewDataListener(func() {
		loaded, _ := w.openedEditor.IsLoaded.Get()
		if loaded {
			loader.Hide()
		} else {
			loader.Show()
		}
	}))

	btns := container.NewBorder(nil, nil,
		widget.NewButtonWithIcon("Save & Exit", theme.DocumentSaveIcon(), func() {
			w.handleSave(editor.Text)
			w.openedEditor.Window.Close()
		}), nil,
		loader,
	)

	c := container.NewBorder(
		container.NewBorder(nil, nil, toolbar, widget.NewLabelWithData(w.stateLabel)),
		btns,
		nil, nil,
		editor)

	return widget.NewSimpleRenderer(c)
}

func (w *FileEditor) handleSave(content string) {
	if err := w.openedEditor.OnSave(content); err != nil {
		w.stateLabel.Set("error")
		dialog.ShowError(err, w.openedEditor.Window)
		return
	}
	w.contentHash = sha256Hex(content)
	w.stateLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05")))
}

func (w *FileEditor) hasChanged(newContent string) bool {
	newHash := sha256Hex(newContent)
	if w.contentHash != newHash {
		w.contentHash = newHash
		return true
	}
	return false
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
