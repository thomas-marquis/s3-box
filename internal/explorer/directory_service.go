package explorer

import (
	"context"
	"fmt"
)

type DirectoryService struct {
	repository Repository
}

func NewDirectoryService(repo Repository) *DirectoryService {
	return &DirectoryService{repository: repo}
}

func (s *DirectoryService) GetRootDirectory() (*Directory, error) {
	return NewDirectory("", nil), nil
}

func (s *DirectoryService) Load(ctx context.Context, d *Directory) error {
	subdirs, files, err := s.repository.ListDirectoryContent(ctx, d)
	if err != nil {
		if err == ErrConnectionNoSet {
			return err
		}
		return fmt.Errorf("impossible to load directory '%s' content: %s", d.Path(), err)
	}
	for _, f := range files {
		d.AddFile(f)
	}
	for _, sd := range subdirs {
		d.AddSubdir(sd)
	}
	d.IsLoaded = true

	return nil
}
