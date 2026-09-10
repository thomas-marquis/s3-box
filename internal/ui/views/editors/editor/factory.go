package editor

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

// Factory defines the interface that hold everything needed to create an editor instance.
type Factory interface {
	// New create a new editor instance. Each editor instance represents a unique opened file.
	New(bus event.Bus, window fyne.Window, file *directory.File) Editor
	// Name returns the unique name of the editor. This name acts like an ID.
	Name() string
	// DisplayLabel returns the label that is visible by the user in the main application.
	DisplayLabel() string
	// DefaultFileRegexpPattern return the default regexp used to decide - given the file path- when to use this editor to open a file.
	// The user can change this pattern from the main application.
	DefaultFileRegexpPattern() string
}
