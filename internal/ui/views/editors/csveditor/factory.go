package csveditor

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type Factory struct{}

func (f *Factory) New(bus event.Bus, window fyne.Window, file *directory.File) editor.Editor {
	return New(bus, window, file)
}

func (f *Factory) Name() string {
	return Name
}

func (f *Factory) DisplayLabel() string {
	return DisplayLabel
}

func (f *Factory) DefaultFileRegexpPattern() string {
	return DefaultPattern
}

func (f *Factory) Capabilities() editor.Capabilities {
	return editor.Capabilities{
		Editable: true,
	}
}

type FactoryRO struct{}

func (f *FactoryRO) New(bus event.Bus, window fyne.Window, file *directory.File) editor.Editor {
	return NewLazy(bus, window, file)
}

func (f *FactoryRO) Name() string {
	return Name + ".ro"
}

func (f *FactoryRO) DisplayLabel() string {
	return DisplayLabel + " (Read-Only)"
}

func (f *FactoryRO) DefaultFileRegexpPattern() string {
	return DefaultPattern
}

func (f *FactoryRO) Capabilities() editor.Capabilities {
	return editor.Capabilities{
		Editable: false,
	}
}
