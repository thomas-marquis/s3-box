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
		if sd.mountPoint.Name() == name {
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
	return newPrev, nil
}

func (p *Preview) AddFile(name string, sizeBytes int, lastModified time.Time) error {
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

type Materializer interface {
	Materialize() event.Event
}

type materializeUploadSkip struct {
	preview     *Preview
	srcBasePath string
}

func NewSkipUploadMaterializer(preview *Preview, srcBasePath string) Materializer {
	return &materializeUploadSkip{preview: preview, srcBasePath: srcBasePath}
}

func (m *materializeUploadSkip) Materialize() event.Event {
	var (
		layerEvts      []event.Event
		currLayerPrevs []*Preview
	)

	currLayerPrevs = append(currLayerPrevs, m.preview)

	for {
		if len(currLayerPrevs) == 0 {
			break
		}

		var evts []event.Event
		for _, prev := range currLayerPrevs {
			for _, f := range prev.files {
				if prev.dir.IsFileExists(f.Name()) {
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
					continue // skip when sub directory already exists
				}
				evts = append(evts, event.New(CreateTriggered{
					ParentDirectory: sd.parent.dir,
					Directory:       sd.dir,
				}))
			}
		}

		layerEvts = append(layerEvts, carrier.NewAll(evts,
			func(evtCarrier event.Event, received []event.Event) event.Event {
				return evtCarrier.NewFollowup(event.ItHappened{})
			},
			nil,
			carrier.WithTimeout(time.Second*120),
		))
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
			return event.New(nil)
		},
		event.New(CreateFileFailed{
			Err:       errors.New("timeout"),
			Directory: nil,
		}),
	)
}
