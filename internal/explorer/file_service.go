package explorer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FileService struct {
	logger     *zap.Logger
	connID     uuid.UUID
	repoFactory FileRepositoryFactory
}

type FileRepositoryFactory func(ctx context.Context, connID uuid.UUID) (S3FileRepository, error)

func NewFileService(logger *zap.Logger, repoFactory FileRepositoryFactory) *FileService {
	return &FileService{
		logger:     logger,
		repoFactory: repoFactory,
		connID:      uuid.Nil,
	}
}

func (s *FileService) SetActiveConnection(connID uuid.UUID) {
	s.connID = connID
}

func (s *FileService) GetContent(ctx context.Context, file *S3File) ([]byte, error) {
	if s.connID == uuid.Nil {
		return nil, ErrConnectionNoSet
	}

	repo, err := s.repoFactory(ctx, s.connID)
	if err != nil {
		return nil, fmt.Errorf("GetContent: %w", err)
	}

	return repo.GetContent(ctx, file.ID)
}

func (s *FileService) DownloadFile(ctx context.Context, file *S3File, dest string) error {
	if s.connID == uuid.Nil {
		return ErrConnectionNoSet
	}

	repo, err := s.repoFactory(ctx, s.connID)
	if err != nil {
		return fmt.Errorf("DownloadFile: %w", err)
	}

	return repo.DownloadFile(ctx, file.ID.String(), dest)
}

func (s *FileService) UploadFile(ctx context.Context, local *LocalFile, remote *S3File) error {
	if s.connID == uuid.Nil {
		return ErrConnectionNoSet
	}

	repo, err := s.repoFactory(ctx, s.connID)
	if err != nil {
		return fmt.Errorf("UploadFile: %w", err)
	}

	return repo.UploadFile(ctx, local, remote)
} 