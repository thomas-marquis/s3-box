package directory

import (
	"errors"

	"github.com/thomas-marquis/it-happened/carrier"
	"github.com/thomas-marquis/it-happened/event"
)

type ChangeType string

const (
	ChangeTypeAdd     ChangeType = "change.directory.add"
	ChangeTypeAddFile ChangeType = "change.directory.addFile"
)

type Change interface {
	Type() ChangeType
	ObjectPath() Path
	Apply() func(directory *Directory) (event.Event, error)
}

type ChangeAdd struct {
	Name string
}

func (c *ChangeAdd) Type() ChangeType {
	return ChangeTypeAdd
}

func (c *ChangeAdd) Apply() func(*Directory) (event.Event, error) {
	return func(dir *Directory) (event.Event, error) {
		return dir.NewSubDirectory(c.Name)
	}
}

type ChangeAddFile struct {
	Name      string
	Overwrite bool
}

func (c *ChangeAddFile) Type() ChangeType {
	return ChangeTypeAddFile
}

func (c *ChangeAddFile) Apply() func(*Directory) (event.Event, error) {
	return func(dir *Directory) (event.Event, error) {
		return dir.NewFile(c.Name, c.Overwrite)
	}
}

type Preview struct {
	dir            *Directory
	changes        []Change
	subDirectories []*Preview
}

func NewPreview(dir *Directory) *Preview {
	p := &Preview{dir: dir}

	if len(dir.SubDirectories()) > 0 {
		subPreviews := make([]*Preview, len(dir.SubDirectories()))
		for i, sd := range dir.SubDirectories() {
			subPreviews[i] = NewPreview(sd)
		}
		p.subDirectories = subPreviews
	}

	return p
}

func (p *Preview) With(change Change) {
	p.changes = append(p.changes, change)
}

func (p *Preview) ApplyAll(
	doneEventFactory func(received []event.Event) event.Event,
	onTimeout event.Event,
	opts ...carrier.Option,
) (event.Event, error) {
	if len(p.changes) == 0 {
		return nil, errors.New("no changes to apply")
	}

	events := make([]event.Event, len(p.changes))
	for i, change := range p.changes {
		evt, err := change.Apply()(p.dir)
		if err != nil {
			return evt, err
		}
		events[i] = evt
	}

	return carrier.NewAll(events, doneEventFactory, onTimeout, opts...), nil
}
