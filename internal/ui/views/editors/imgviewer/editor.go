package imgviewer

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type Editor struct {
	*editor.Base
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	e := &Editor{
		Base: editor.NewBase(bus, w, file),
	}
	e.ExtendBaseEditor(e)
	return e
}

func (e *Editor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}
