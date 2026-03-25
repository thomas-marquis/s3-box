package fileeditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type State struct {
	Window fyne.Window
	File   *directory.File

	Content  binding.String
	IsLoaded binding.Bool
	ErrorMsg binding.String

	Bus event.Bus
}
