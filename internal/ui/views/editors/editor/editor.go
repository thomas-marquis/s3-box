package editor

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type Initializer func(bus event.Bus, window fyne.Window, file *directory.File) Editor

type Editor interface {
	Window() fyne.Window
	File() *directory.File
	CreateWidget() fyne.CanvasObject
	OnLoaded(fileContent directory.FileContent, err error)
	OnSaved(newContent string, err error)
}

// Closable represent an editor that can be closed properly by the main application.
type Closable interface {
	// BeforeClose is called right before the editor is closed externally (from click on the cross button or via the main application).
	// The callback is called with true if the editor is ready to be closed (modifications saved, etc.), or returns false otherwise.
	// This method MUST NOT call Window.Close().
	BeforeClose(cb func(ready bool))
}

type Base struct {
	window fyne.Window
	file   *directory.File
}

func NewBase(window fyne.Window, file *directory.File) Base {
	return Base{
		window: window,
		file:   file,
	}
}

func (b *Base) Window() fyne.Window {
	return b.window
}

func (b *Base) File() *directory.File {
	return b.file
}

func (b *Base) Close() {
	b.window.Close()
}
