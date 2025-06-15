package explorer_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/thomas-marquis/s3-box/internal/explorer"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connections"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func getDirectoryServiceMocks(t *testing.T) (*gomock.Controller, *mocks_explorer.MockS3DirectoryRepository, *mocks_explorer.MockS3FileRepository, *mocks_connection.MockConnectionService) {
	ctrl := gomock.NewController(t)
	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
	connSvc := mocks_connection.NewMockConnectionService(ctrl)
	return ctrl, dirRepo, fileRepo, connSvc
}

func Test_GetRootDirectory_ShouldReturnRootDirectory(t *testing.T) {
	// Given
	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return nil, nil
		},
		connSvc,
	)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(1)

	expectedDir := &explorer.S3Directory{
		ID:   explorer.RootDirID,
		Name: "",
	}
	ctx := context.TODO()

	dirRepo.EXPECT().
		GetByID(ctx, explorer.RootDirID).
		Return(expectedDir, nil).
		Times(1)

	// When
	dir, err := svc.GetRootDirectory(ctx)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, expectedDir, dir)
}

func Test_GetRootDirectory_ShouldReturnErrorWhenNoActiveConnection(t *testing.T) {
	// Given
	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return nil, nil
		},
		connSvc,
	)

	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(uuid.Nil, explorer.ErrConnectionNoSet).
		Times(1)

	ctx := context.TODO()

	// When
	dir, err := svc.GetRootDirectory(ctx)

	// Then
	assert.Equal(t, explorer.ErrConnectionNoSet, err)
	assert.Nil(t, dir)
}

func Test_GetDirectoryByID_ShouldReturnDirectory(t *testing.T) {
	// Given
	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return nil, nil
		},
		connSvc,
	)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(1)

	dirID := explorer.S3DirectoryID("path/to/dir")
	expectedDir := &explorer.S3Directory{
		ID:   dirID,
		Name: "dir",
	}
	ctx := context.TODO()

	dirRepo.EXPECT().
		GetByID(ctx, dirID).
		Return(expectedDir, nil).
		Times(1)

	// When
	dir, err := svc.GetDirectoryByID(ctx, dirID)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, expectedDir, dir)
}

func Test_GetDirectoryByID_ShouldReturnErrorWhenNoActiveConnection(t *testing.T) {
	// Given
	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return nil, nil
		},
		connSvc,
	)

	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(uuid.Nil, explorer.ErrConnectionNoSet).
		Times(1)

	dirID := explorer.S3DirectoryID("path/to/dir")
	ctx := context.TODO()

	// When
	dir, err := svc.GetDirectoryByID(ctx, dirID)

	// Then
	assert.Equal(t, explorer.ErrConnectionNoSet, err)
	assert.Nil(t, dir)
}

func Test_DeleteFile_ShouldDeleteFile(t *testing.T) {
	// Given
	ctrl, dirRepo, fileRepo, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		connSvc,
	)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(2)

	dirID := explorer.S3DirectoryID("path/to/dir")
	fileID := explorer.S3FileID("file.txt")
	dir := &explorer.S3Directory{
		ID:   dirID,
		Name: "dir",
		Files: []*explorer.S3File{
			{
				ID:   fileID,
				Name: "file.txt",
			},
		},
	}
	ctx := context.TODO()

	dirRepo.EXPECT().
		Save(ctx, dir).
		Return(nil).
		Times(1)

	fileRepo.EXPECT().
		DeleteFile(ctx, fileID).
		Return(nil).
		Times(1)

	// When
	err := svc.DeleteFile(ctx, dir, fileID)

	// Then
	assert.NoError(t, err)
	assert.Empty(t, dir.Files)
}

func Test_DeleteFile_ShouldReturnErrorWhenFileNotInDirectory(t *testing.T) {
	// Given
	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return nil, nil
		},
		connSvc,
	)

	dirID := explorer.S3DirectoryID("path/to/dir")
	fileID := explorer.S3FileID("file.txt")
	dir := &explorer.S3Directory{
		ID:   dirID,
		Name: "dir",
		Files: []*explorer.S3File{
			{
				ID:   explorer.S3FileID("other.txt"),
				Name: "other.txt",
			},
		},
	}
	ctx := context.TODO()

	// When
	err := svc.DeleteFile(ctx, dir, fileID)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to directory")
	assert.Len(t, dir.Files, 1)
}

func Test_DeleteFile_ShouldRestoreDirectoryWhenSavingFails(t *testing.T) {
	// Given
	ctrl, dirRepo, fileRepo, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		connSvc,
	)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(1)

	dirID := explorer.S3DirectoryID("path/to/dir")
	fileID := explorer.S3FileID("file.txt")
	file := &explorer.S3File{
		ID:   fileID,
		Name: "file.txt",
	}
	dir := &explorer.S3Directory{
		ID:    dirID,
		Name:  "dir",
		Files: []*explorer.S3File{file},
	}
	ctx := context.TODO()

	dirRepo.EXPECT().
		Save(ctx, dir).
		Return(fmt.Errorf("save error")).
		Times(1)

	// When
	err := svc.DeleteFile(ctx, dir, fileID)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving directory")
	assert.Len(t, dir.Files, 1)
	assert.Equal(t, file, dir.Files[0])
}

func Test_DeleteFile_ShouldRestoreDirectoryWhenGettingFileRepositoryFails(t *testing.T) {
	// Given
	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return nil, fmt.Errorf("file repo error")
		},
		connSvc,
	)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(2)

	dirID := explorer.S3DirectoryID("path/to/dir")
	fileID := explorer.S3FileID("file.txt")
	file := &explorer.S3File{
		ID:   fileID,
		Name: "file.txt",
	}
	dir := &explorer.S3Directory{
		ID:    dirID,
		Name:  "dir",
		Files: []*explorer.S3File{file},
	}
	ctx := context.TODO()

	dirRepo.EXPECT().
		Save(ctx, dir).
		Return(nil).
		Times(2)

	// When
	err := svc.DeleteFile(ctx, dir, fileID)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting file repository")
	assert.Len(t, dir.Files, 1)
	assert.Equal(t, file, dir.Files[0])
}

func Test_DeleteFile_ShouldRestoreDirectoryWhenDeletingFileFails(t *testing.T) {
	// Given
	ctrl, dirRepo, fileRepo, connSvc := getDirectoryServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		connSvc,
	)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(2)

	dirID := explorer.S3DirectoryID("path/to/dir")
	fileID := explorer.S3FileID("file.txt")
	file := &explorer.S3File{
		ID:   fileID,
		Name: "file.txt",
	}
	dir := &explorer.S3Directory{
		ID:    dirID,
		Name:  "dir",
		Files: []*explorer.S3File{file},
	}
	ctx := context.TODO()

	dirRepo.EXPECT().
		Save(ctx, dir).
		Return(nil).
		Times(2)

	fileRepo.EXPECT().
		DeleteFile(ctx, fileID).
		Return(fmt.Errorf("delete error")).
		Times(1)

	// When
	err := svc.DeleteFile(ctx, dir, fileID)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error deleting file")
	assert.Len(t, dir.Files, 1)
	assert.Equal(t, file, dir.Files[0])
}
