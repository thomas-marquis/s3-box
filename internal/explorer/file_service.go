package explorer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"go.uber.org/zap"
)

type FileService struct {
	logger      *zap.Logger
	repoFactory FileRepositoryFactory
	connSvc     connection.ConnectionService
}

type FileRepositoryFactory func(ctx context.Context, connID uuid.UUID) (S3FileRepository, error)

func NewFileService(
	logger *zap.Logger,
	repoFactory FileRepositoryFactory,
	connSvc connection.ConnectionService,
) *FileService {
	return &FileService{
		logger:      logger,
		repoFactory: repoFactory,
		connSvc:     connSvc,
	}
}

func (s *FileService) GetContent(ctx context.Context, file *S3File) ([]byte, error) {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return nil, err
	}

	return repo.GetContent(ctx, file.ID)
}

func (s *FileService) DownloadFile(ctx context.Context, file *S3File, dest string) error {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return err
	}

	return repo.DownloadFile(ctx, file.ID.String(), dest)
}

func (s *FileService) UploadFile(ctx context.Context, local *LocalFile, remote *S3File) error {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return err
	}

	return repo.UploadFile(ctx, local, remote)
}

func (s *FileService) getActiveRepository(ctx context.Context) (S3FileRepository, error) {
	connId, err := s.connSvc.GetActiveConnectionID(ctx)
	if connId == uuid.Nil || err == ErrConnectionNoSet {
		return nil, ErrConnectionNoSet
	}
	if err != nil {
		return nil, fmt.Errorf("error whe getting file repository: %w", err)
	}
	return s.repoFactory(ctx, connId)
}

