package directory

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/thomas-marquis/it-happened/carrier"
	"github.com/thomas-marquis/it-happened/event"
)

type MaterializeStrategy int

func (s MaterializeStrategy) String() string {
	switch s {
	case MaterializeSkip:
		return "Skip"
	case MaterializeReplace:
		return "Replace"
	default:
		panic("invalid materialize strategy")
	}
}

const (
	MaterializeSkip = iota
	MaterializeReplace
)

type Preview struct {
	mountPoint *Directory
	dir        *Directory

	parent              *Preview
	children            []*Preview
	files               []*File
	availableStrategies map[MaterializeStrategy]struct{}
}

func newPreview(mount, dir *Directory) *Preview {
	p := &Preview{
		mountPoint:          mount,
		dir:                 dir,
		files:               make([]*File, 0),
		children:            make([]*Preview, 0),
		availableStrategies: make(map[MaterializeStrategy]struct{}),
	}
	p.availableStrategies[MaterializeSkip] = struct{}{} // default strategy
	return p
}

func (p *Preview) AddSubDirectory(name string) (*Preview, error) {
	for _, sd := range p.children {
		if sd.dir.Name() == name {
			return nil, errors.New("sub directory preview already exists in the preview")
		}
	}

	if err := validateName(name, p.dir.Path()); err != nil {
		return nil, err
	}

	var subDir *Directory

	if sd, err := p.dir.GetSubDirectoryByName(name); errors.Is(err, ErrNotFound) {
		subDir, err = New(p.dir.ConnectionID(), name, p.dir)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		subDir = sd
	}

	newPrev := newPreview(p.mountPoint, subDir)
	p.children = append(p.children, newPrev)
	newPrev.parent = p
	return newPrev, nil
}

func (p *Preview) AddFile(name string, sizeBytes uint64, lastModified time.Time) error {
	for _, f := range p.files {
		if f.Name().String() == name {
			return fmt.Errorf("file %s already exists in the preview", name)
		}
	}
	fn, err := NewFileName(name)
	if err != nil {
		return err
	}

	if lastModified.IsZero() {
		lastModified = time.Now()
	}
	f := &File{
		name:         fn,
		parent:       p.dir,
		sizeBytes:    sizeBytes,
		lastModified: lastModified,
	}
	p.files = append(p.files, f)

	if p.dir.IsFileExists(fn) {
		p.availableStrategies[MaterializeReplace] = struct{}{}
	}
	return nil
}

func (p *Preview) Directory() *Directory {
	return p.dir
}

func (p *Preview) MountPoint() *Directory {
	return p.mountPoint
}

func (p *Preview) Files() []*File {
	return p.files
}

func (p *Preview) Children() []*Preview {
	return p.children
}

func (p *Preview) AvailableStrategies() []MaterializeStrategy {
	var strats []MaterializeStrategy
	for s := range p.availableStrategies {
		strats = append(strats, s)
	}
	return strats
}

func (p *Preview) FileStatus(strategy MaterializeStrategy, fileName string) (string, string, error) {
	var previewed *File
	for _, f := range p.files {
		if f.Name().String() == fileName {
			previewed = f
			break
		}
	}
	if previewed == nil {
		return "", "", ErrNotFound
	}
	actual, err := p.dir.GetFileByName(FileName(fileName))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", "", err
	}

	if actual == nil {
		return "New", "", nil
	}
	switch strategy {
	case MaterializeSkip:
		return "Skipped", fmt.Sprintf("The file '%s' already exists and will be kept untouched", actual.Name()), nil
	case MaterializeReplace:
		return "Replaced", fmt.Sprintf("The file '%s' already exists and will be replaced by the new version (previous: %dKB; new: %dKB)",
			actual.Name(), actual.SizeBytes()/1024, previewed.SizeBytes()/1024), nil
	}
	panic("invalid materialize strategy")
}

func (p *Preview) DirStatus() string {
	if p.dir.Parent().IsSubDirectoryExists(p.dir.Name()) {
		return ""
	}
	return "New"
}

func (p *Preview) GetByPath(path Path) (*Preview, error) {
	if p.dir.Path() == path {
		return p, nil
	}
	for _, child := range p.children {
		if found, err := child.GetByPath(path); err == nil {
			return found, nil
		}
	}
	return nil, ErrNotFound
}

type Materializer interface {
	Materialize(strategy MaterializeStrategy) event.Event
}

type materializeUpload struct {
	preview     *Preview
	srcBasePath string
}

func NewUploadMaterializer(preview *Preview, srcBasePath string) Materializer {
	return &materializeUpload{preview: preview, srcBasePath: srcBasePath}
}

func (m *materializeUpload) Materialize(strategy MaterializeStrategy) event.Event {
	var (
		layerEvts      []event.Event
		currLayerPrevs []*Preview
	)

	currLayerPrevs = append(currLayerPrevs, m.preview)

	for len(currLayerPrevs) > 0 {
		var evts []event.Event
		for _, prev := range currLayerPrevs {
			for _, f := range prev.files {
				if strategy == MaterializeSkip && prev.dir.IsFileExists(f.Name()) {
					continue // skip when file already exists
				}

				fileRelPath, err := prev.dir.Path().RelativeTo(prev.mountPoint.Path())
				if err != nil {
					panic(err) // should never happen
				}
				uploadPath := filepath.Join(m.srcBasePath, fileRelPath.String(), f.Name().String())

				evts = append(evts, event.New(UploadFileTriggered{
					Directory: prev.dir,
					SrcPath:   uploadPath,
				}))
			}

			for _, sd := range prev.children {
				if prev.dir.IsSubDirectoryExists(sd.dir.Name()) {
					continue // skip when subdirectory already exists, whatever the strategy
				}
				evts = append(evts, event.New(CreateTriggered{
					ParentDirectory: sd.parent.dir,
					Directory:       sd.dir,
				}))
			}
		}

		if len(evts) != 0 {
			layerEvts = append(layerEvts, carrier.NewAll(evts,
				func(evtCarrier event.Event, received []event.Event) event.Event {
					return evtCarrier.NewFollowup(event.ItHappened{})
				},
				nil,
				carrier.WithTimeout(time.Second*120),
			))
		}
		tmpLay := make([]*Preview, 0)
		for _, sd := range currLayerPrevs {
			if sd.children == nil {
				continue
			}
			tmpLay = append(tmpLay, sd.children...)
		}
		currLayerPrevs = tmpLay
	}

	return carrier.NewSequence(
		layerEvts,
		func(evtCarrier event.Event, received []event.Event) event.Event {
			return evtCarrier.NewFollowup(event.ItHappened{})
		},
		event.New(CreateFileFailed{
			Err:       errors.New("timeout"),
			Directory: nil,
		}),
	)
}

type materializeLoad struct {
	preview             *Preview
	doneEventPayload    event.Payload
	timeoutEventPayload event.Payload
}

func NewLoadMaterializer(preview *Preview, doneEventPayload, timeoutEventPayload event.Payload) Materializer {
	return &materializeLoad{preview: preview, doneEventPayload: doneEventPayload, timeoutEventPayload: timeoutEventPayload}
}

func (m *materializeLoad) Materialize(MaterializeStrategy) event.Event {
	return m.makeNextEvent(nil, []*Directory{m.preview.Directory()})
}

func (m *materializeLoad) makeNextEvent(carrierEvt event.Event, dirs []*Directory) event.Event {
	var nextEvts []event.Event

	for _, dir := range dirs {
		_, err := m.preview.GetByPath(dir.Path())
		if err != nil {
			continue
		}
		evt, err := dir.Load()
		if err != nil {
			continue
		}
		nextEvts = append(nextEvts, evt)
	}

	if len(nextEvts) == 0 {
		if carrierEvt != nil {
			return carrierEvt.NewFollowup(m.doneEventPayload)
		}
		return event.New(m.doneEventPayload)
	}

	return carrier.NewAll(nextEvts,
		func(evtCarrier event.Event, received []event.Event) event.Event {
			var nexDirs []*Directory
			for _, rcv := range received {
				switch pl := rcv.Payload().(type) {
				case LoadSucceeded:
					for _, sd := range pl.Directory.SubDirectories() {
						if _, err := m.preview.GetByPath(sd.Path()); err == nil {
							nexDirs = append(nexDirs, sd)
						}
					}
				case LoadFailed:
					// ignoring failures, for now...
					continue
				}
			}
			return m.makeNextEvent(evtCarrier, nexDirs)
		},
		event.New(m.timeoutEventPayload),
	)
}
