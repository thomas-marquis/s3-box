package explorer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"go.uber.org/zap"
)

type FileService interface {
	GetContent(ctx context.Context, file *S3File) ([]byte, error)
	DownloadFile(ctx context.Context, file *S3File, dest string) error
	UploadFile(ctx context.Context, local *LocalFile, remote *S3File) error
}

type fileServiceImpl struct {
	logger      *zap.Logger
	repoFactory FileRepositoryFactory
	connSvc     connection.ConnectionService
}

var _ FileService = &fileServiceImpl{}

type FileRepositoryFactory func(ctx context.Context, connID uuid.UUID) (S3FileRepository, error)

func NewFileService(
	logger *zap.Logger,
	repoFactory FileRepositoryFactory,
	connSvc connection.ConnectionService,
) *fileServiceImpl {
	return &fileServiceImpl{
		logger:      logger,
		repoFactory: repoFactory,
		connSvc:     connSvc,
	}
}

func (s *fileServiceImpl) GetContent(ctx context.Context, file *S3File) ([]byte, error) {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return nil, err
	}

	return repo.GetContent(ctx, file.ID)
}

func (s *fileServiceImpl) DownloadFile(ctx context.Context, file *S3File, dest string) error {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return err
	}

	return repo.DownloadFile(ctx, file.ID.String(), dest)
}

func (s *fileServiceImpl) UploadFile(ctx context.Context, local *LocalFile, remote *S3File) error {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return err
	}

	return repo.UploadFile(ctx, local, remote)
}

func (s *fileServiceImpl) getActiveRepository(ctx context.Context) (S3FileRepository, error) {
	connId, err := s.connSvc.GetActiveConnectionID(ctx)
	if connId == uuid.Nil || err == ErrConnectionNoSet {
		return nil, ErrConnectionNoSet
	}
	if err != nil {
		return nil, fmt.Errorf("error whe getting file repository: %w", err)
	}
	return s.repoFactory(ctx, connId)
}
