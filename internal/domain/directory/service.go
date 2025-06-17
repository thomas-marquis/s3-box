package directory

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type Service interface {
	GetRoot(ctx context.Context) (*Directory, error)
	DownloadFile(ctx context.Context, dir *Directory, fileName FileName, destFullPath string) error
	UploadFile(ctx context.Context, srcFullPath string, destDir *Directory) error
}

type serviceImpl struct {
	repo Repository
}

var _ Service = &serviceImpl{}

func NewService(repo Repository) *serviceImpl {
	return &serviceImpl{repo}
}

func (s *serviceImpl) GetRoot(ctx context.Context) (*Directory, error) {
	dir, err := s.repo.GetByPath(ctx, RootPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("root directory not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("error getting root directory: %w", errors.Join(ErrTechnical, err))
	}
	return dir, nil
}

func (s *serviceImpl) DownloadFile(ctx context.Context, dir *Directory, fileName FileName, destFullPath string) error {
	if !dir.IsFileExists(fileName) {
		return ErrNotFound
	}

	f, err := dir.GetFile(fileName)
	if err != nil {
		return err
	}

	if err := s.repo.DownloadFile(ctx, f, destFullPath); err != nil {
		return fmt.Errorf("error downloading file %s: %w", f.FullPath(), errors.Join(ErrTechnical, err))
	}
	return nil
}

func (s *serviceImpl) UploadFile(ctx context.Context, srcFullPath string, destDir *Directory) error {
	fileName := filepath.Base(srcFullPath)
	newFile, err := destDir.NewFile(fileName)
	if err != nil {
		return err
	}

	if err := s.repo.UploadFile(ctx, srcFullPath, newFile); err != nil {
		destDir.RemoveFile(newFile.Name())
		return fmt.Errorf("error uploading file %s: %w", newFile.FullPath(), errors.Join(ErrTechnical, err))
	}

	return nil
}
