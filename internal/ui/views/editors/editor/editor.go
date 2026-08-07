package editor

import (
	"encoding/json"
	"errors"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

var (
	shortcutQuit = desktop.CustomShortcut{
		KeyName:  fyne.KeyQ,
		Modifier: fyne.KeyModifierControl,
	}
)

type Initializer func(bus event.Bus, window fyne.Window, file *directory.File) Editor

type Editor interface {
	Window() fyne.Window
	File() *directory.File
	CreateWidget() fyne.CanvasObject
}

type Base struct {
	sync.Mutex

	window fyne.Window
	file   *directory.File

	Sub          *event.Subscriber
	StatusLabel  binding.String
	Err          binding.Item[error]
	IsLoading    binding.Bool
	ConfirmClose func(onConfirm func(confirmed bool))
	Bus          event.Bus
	Content      directory.FileContent
}

func NewBase(bus event.Bus, window fyne.Window, file *directory.File) Base {
	e := Base{
		window:       window,
		file:         file,
		StatusLabel:  binding.NewString(),
		IsLoading:    binding.NewBool(),
		Err:          binding.NewItem(errors.Is),
		ConfirmClose: func(onConfirm func(confirmed bool)) {},
		Bus:          bus,
	}

	return e
}

func (b *Base) ExtendBaseEditor(e Editor) {
	b.Sub = b.Bus.Subscribe(forCurrentEditor{Editor: e}).
		DetachOn(event.Is(ClosedType))

	b.window.Canvas().AddShortcut(&shortcutQuit, func(fyne.Shortcut) {
		b.Bus.Publish(event.New(CloseRequested{
			Editor: e,
		}))
	})
}

func (b *Base) Window() fyne.Window {
	return b.window
}

func (b *Base) File() *directory.File {
	return b.file
}

func (b *Base) MarshalJSON() ([]byte, error) {
	status, _ := b.StatusLabel.Get()
	loading, _ := b.IsLoading.Get()
	err, _ := b.Err.Get()

	return json.Marshal(struct {
		File        *directory.File
		StatusLabel string
		IsLoading   bool
		Err         error
	}{
		File:        b.file,
		StatusLabel: status,
		IsLoading:   loading,
		Err:         err,
	})
}

func (b *Base) IsLoaded() bool {
	return b.Content != nil
}

func (b *Base) SetContent(content directory.FileContent) {
	b.Lock()
	defer b.Unlock()

	b.Content = content
}
