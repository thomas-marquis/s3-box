package editor

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type Editor interface {
	Window() fyne.Window
	File() *directory.File
	CreateWidget() fyne.CanvasObject
	OnLoaded(fileContent directory.FileContent, err error)
	OnSaved(newContent string, err error)
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
