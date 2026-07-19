package editor

import (
	"io"

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
	// The callback must be called with ready=true if the editor is ready to be closed (modifications saved, etc.), or with false otherwise.
	// This method MUST NOT call Window.Close().
	BeforeClose(cb func(ready bool))

	// SetCloser register an io.Closer object that the editor can use to get closed itself.
	// This method is called before all others on the editor initialization.
	SetCloser(closer io.Closer)
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
