package widget

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

	win fyne.Window

	data     binding.String
	errorMsg binding.String
	isLoaded binding.Bool

	updated bool

	OnSave func(fileContent string) error
}

func NewFileEditor(data, errorMsg binding.String, isLoaded binding.Bool, win fyne.Window) *FileEditor {
	w := &FileEditor{
		data:     data,
		errorMsg: errorMsg,
		isLoaded: isLoaded,
		OnSave:   func(string) error { return nil },
		win:      win,
	}
	w.ExtendBaseWidget(w)

	//var contentLoaded bool
	//data.AddListener(binding.NewDataListener(func() {
	//	loaded, _ := isLoaded.Get()
	//	if loaded {
	//		if !contentLoaded {
	//			contentLoaded = true
	//		} else {
	//			w.updated = true
	//		}
	//	}
	//}))

	win.SetCloseIntercept(func() {
		if w.updated {
			dialog.ShowConfirm("Discard changes?", "Do you want to discard your changes?", func(confirmed bool) {
				if confirmed {
					w.updated = false
					win.Close()
				}
			}, win)
		} else {
			win.Close()
		}
	})

	errorMsg.AddListener(binding.NewDataListener(func() {
		msg, _ := errorMsg.Get()
		if msg == "" {
			return
		}
		dialog.ShowError(errors.New(msg), win)
	}))
	return w
}

func (w *FileEditor) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	stateLabel := binding.NewString()

	editor := newTextEditorEntry(func(s string) {
		if err := w.OnSave(s); err != nil {
			stateLabel.Set("error")
			dialog.ShowError(err, w.win)
		}
		stateLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05")))
	})
	editor.Bind(w.data)

	editor.OnChanged = func(val string) {
		loaded, _ := w.isLoaded.Get()
		if loaded {
			w.updated = true
		}
	}

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			if err := w.OnSave(editor.Text); err != nil {
				stateLabel.Set("error")
				dialog.ShowError(err, w.win)
				return
			}
			stateLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05")))
		}),
	)

	loader := widget.NewProgressBarInfinite()
	loader.Hide()
	w.isLoaded.AddListener(binding.NewDataListener(func() {
		loaded, _ := w.isLoaded.Get()
		if loaded {
			loader.Hide()
		} else {
			loader.Show()
		}
	}))

	btns := container.NewBorder(nil, nil,
		widget.NewButtonWithIcon("Save & Exit", theme.DocumentSaveIcon(), func() {
			if err := w.OnSave(editor.Text); err != nil {
				stateLabel.Set("error")
				dialog.ShowError(err, w.win)
				return
			}
			w.win.Close()
		}), nil,
		loader,
	)

	c := container.NewBorder(
		container.NewBorder(nil, nil, toolbar, widget.NewLabelWithData(stateLabel)),
		btns,
		nil, nil,
		editor)

	return widget.NewSimpleRenderer(c)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
