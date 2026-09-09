package editor

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

// Factory defines the interface that hold everything needed to create an editor instance.
type Factory interface {
	New(bus event.Bus, window fyne.Window, file *directory.File) Editor
	Name() string
	DisplayLabel() string
	DefaultFileRegexpPattern() string
}
