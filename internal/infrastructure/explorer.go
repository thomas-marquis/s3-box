package infrastructure

import (
	"context"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"

	"go.uber.org/zap"
)

type ExplorerRepositoryImpl struct {
	log *zap.SugaredLogger
	s3  *s3Client

	conn *connection.Connection
}

var _ explorer.Repository = &ExplorerRepositoryImpl{}

func NewExplorerRepositoryImpl(logger *zap.Logger, defaultConn *connection.Connection) (*ExplorerRepositoryImpl, error) {
	log := logger.Sugar()
	var remoteSrc *s3Client
	var err error
	if defaultConn != nil {
		remoteSrc, err = newS3Client(log, defaultConn.AccessKey, defaultConn.SecretKey, defaultConn.Server, defaultConn.BucketName, defaultConn.Region, defaultConn.UseTls)
		if err != nil {
			return nil, fmt.Errorf("NewExplorerRepositoryImpl: %w", err)
		}
	}

	return &ExplorerRepositoryImpl{
		log: log,
		s3:  remoteSrc,
	}, nil
}

func (r *ExplorerRepositoryImpl) ListDirectoryContent(ctx context.Context, dir *explorer.S3Directory) ([]*explorer.S3Directory, []*explorer.S3File, error) {
	if r.s3 == nil {
		return nil, nil, explorer.ErrConnectionNoSet
	}

	dirs, files, err := r.s3.GetDirectoriesAndFileByPath(ctx, dir)
	if err != nil {
		r.log.Errorf("error getting directories and files: %v\n", err)
		return nil, nil, fmt.Errorf("ListDirectoryContent: %w", err)
	}

	return dirs, files, nil
}

func (r *ExplorerRepositoryImpl) GetFileContent(ctx context.Context, file *explorer.S3File) ([]byte, error) {
	if r.s3 == nil {
		return nil, explorer.ErrConnectionNoSet
	}

	return r.s3.GetFileContent(ctx, file)
}

func (r *ExplorerRepositoryImpl) SetConnection(ctx context.Context, c *connection.Connection) error {
	r.conn = c
	newRemote, err := newS3Client(r.log, c.AccessKey, c.SecretKey, c.Server, c.BucketName, c.Region, c.UseTls)
	if err != nil {
		return fmt.Errorf("SetConnection: %w", err)
	}
	r.s3 = newRemote
	return nil
}

func (r *ExplorerRepositoryImpl) DownloadFile(ctx context.Context, file *explorer.S3File, dest string) error {
	if r.s3 == nil {
		return explorer.ErrConnectionNoSet
	}

	fileKey := file.Path()
	if err := r.s3.DownloadFile(ctx, fileKey, dest); err != nil {
		r.log.Errorf("error downloading file: %v\n", err)
		return fmt.Errorf("DownloadFile: %w", err)
	}

	return nil
}

func (r *ExplorerRepositoryImpl) UploadFile(ctx context.Context, local *explorer.LocalFile, remote *explorer.S3File) error {
	if r.s3 == nil {
		return explorer.ErrConnectionNoSet
	}

	if err := r.s3.UploadFile(ctx, local, remote); err != nil {
		r.log.Errorf("error uploading file: %v\n", err)
		return fmt.Errorf("UploadFile: %w", err)
	}

	return nil
}

func (r *ExplorerRepositoryImpl) DeleteFile(ctx context.Context, remote *explorer.S3File) error {
	return r.s3.DeleteFile(ctx, remote)
}
