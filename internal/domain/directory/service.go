package directory

import (
	"context"
	"errors"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

type Service interface {
	GetRoot(ctx context.Context, connId connection_deck.ConnectionID) (*Directory, error)
	DownloadFile(ctx context.Context, connId connection_deck.ConnectionID, dir *Directory, fileName FileName, destFullPath string) error
	UploadFile(ctx context.Context, connId connection_deck.ConnectionID, srcFullPath string, destDir *Directory) error
}

type serviceImpl struct {
	repo      Repository
	publisher EventPublisher
}

var _ Service = &serviceImpl{}

func NewService(repo Repository, publisher EventPublisher) Service {
	return &serviceImpl{repo, publisher}
}

func (s *serviceImpl) GetRoot(ctx context.Context, connId connection_deck.ConnectionID) (*Directory, error) {
	dir, err := s.repo.GetByPath(ctx, connId, RootPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("root directory not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("error getting root directory: %w", errors.Join(ErrTechnical, err))
	}
	return dir, nil
}

func (s *serviceImpl) UploadFile(
	ctx context.Context,
	connId connection_deck.ConnectionID,
	srcFullPath string,
	destDir *Directory,
) error {
	evt, err := destDir.UploadFile(srcFullPath)
	if err != nil {
		return fmt.Errorf("error uploading fileObj %s: %w", srcFullPath, err)
	}
	s.publisher.Publish(evt)

	return nil
}
