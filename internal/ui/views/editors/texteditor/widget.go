package texteditor

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
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/fileeditor"
)

type textContentEntry struct {
	widget.Entry

	onValidate func(string)
}

var (
	_ fyne.Shortcutable = (*textContentEntry)(nil)
)

func (e *textContentEntry) TypedShortcut(s fyne.Shortcut) {
	if val, ok := s.(*desktop.CustomShortcut); ok {
		if val.KeyName == fyne.KeyS && val.Modifier == fyne.KeyModifierControl {
			e.onValidate(e.Text)
		}
	} else {
		e.Entry.TypedShortcut(s)
	}
}

func newTextEditorEntry(onValidate func(string)) *textContentEntry {
	e := &textContentEntry{
		Entry: widget.Entry{
			MultiLine: true,
			Wrapping:  fyne.TextWrap(fyne.TextTruncateClip),
		},
		onValidate: onValidate,
	}
	e.ExtendBaseWidget(e)
	return e
}

type TextEditor struct {
	widget.BaseWidget

	state                *fileeditor.State
	contentHash          string
	stateLabel           binding.String
	shouldCloseWhenSaved bool
}

func NewTextEditor(state *fileeditor.State) *TextEditor {
	w := &TextEditor{
		state:      state,
		stateLabel: binding.NewString(),
	}
	w.ExtendBaseWidget(w)

	once := sync.Once{}

	state.Content.AddListener(binding.NewDataListener(func() {
		loaded, _ := state.IsLoaded.Get()
		if !loaded {
			return
		}
		once.Do(func() {
			val, _ := state.Content.Get()
			w.contentHash = sha256Hex(val)
		})
	}))

	state.Window.SetCloseIntercept(func() {
		val, _ := state.Content.Get()
		if w.hasChanged(val) {
			dialog.ShowConfirm("Discard changes?", "Do you want to discard your changes?", func(confirmed bool) {
				if confirmed {
					state.Window.Close()
				}
			}, state.Window)
		} else {
			state.Window.Close()
		}
	})

	state.ErrorMsg.AddListener(binding.NewDataListener(func() {
		msg, _ := state.ErrorMsg.Get()
		if msg == "" {
			return
		}
		dialog.ShowError(errors.New(msg), state.Window)
	}))

	w.state.Bus.Subscribe().
		On(event.Is(fileeditor.SaveEventType), func(e event.Event) {
			evt := e.(fileeditor.SaveEvent)
			if !evt.File.Is(state.File) {
				return
			}

			state.IsLoaded.Set(false)     // nolint:errcheck
			w.stateLabel.Set("Saving...") // nolint:errcheck
		}).
		On(event.Is(fileeditor.SaveEventType.AsSuccess()), func(e event.Event) {
			evt := e.(fileeditor.SaveSuccessEvent)
			if !evt.File.Is(state.File) {
				return
			}
			w.contentHash = sha256Hex(evt.Content)
			w.stateLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05"))) // nolint:errcheck
			state.IsLoaded.Set(true)                                                 // nolint:errcheck
			if w.shouldCloseWhenSaved {
				state.Window.Close()
			}
		}).
		On(event.Is(fileeditor.SaveEventType.AsFailure()), func(e event.Event) {
			evt := e.(fileeditor.SaveFailureEvent)
			if !evt.File.Is(state.File) {
				return
			}
			state.IsLoaded.Set(true)
			w.stateLabel.Set("error") // nolint:errcheck
			dialog.ShowError(evt.Error(), w.state.Window)
			w.shouldCloseWhenSaved = false
		}).
		ListenWithWorkers(1)

	return w
}

func (w *TextEditor) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	editor := newTextEditorEntry(w.save)
	editor.Bind(w.state.Content)

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			w.save(editor.Text)
		}),
	)

	loader := widget.NewProgressBarInfinite()
	loader.Hide()
	w.state.IsLoaded.AddListener(binding.NewDataListener(func() {
		loaded, _ := w.state.IsLoaded.Get()
		if loaded {
			loader.Hide()
		} else {
			loader.Show()
		}
	}))

	btns := container.NewBorder(nil, nil,
		widget.NewButtonWithIcon("Save & Exit", theme.DocumentSaveIcon(), func() {
			w.save(editor.Text)
			w.shouldCloseWhenSaved = true
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

func (w *TextEditor) save(content string) {
	w.state.Bus.Publish(fileeditor.NewSaveEvent(w.state.File, content))
}

func (w *TextEditor) hasChanged(newContent string) bool {
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
