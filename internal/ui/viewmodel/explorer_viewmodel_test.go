package viewmodel_test

import (
	"context"
	"testing"

	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func Test_RefreshDir_ShouldRefreshDirectoryContent(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
	connSvc := mocks_connection.NewMockConnectionService(ctrl)
	connRepo := mocks_connection.NewMockRepository(ctrl)
	dirSvc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		connSvc,
	)
	vm := viewmodel.NewExplorerViewModel(dirSvc, connRepo, nil, viewmodel.NewSettingsViewModel(nil))

	dirID := explorer.S3DirectoryID("/test")
	newDir := &explorer.S3Directory{
		ID:   dirID,
		Name: "test",
		Files: []*explorer.S3File{
			{ID: "test/new.txt", Name: "new.txt"},
		},
		SubDirectoriesIDs: []explorer.S3DirectoryID{"test/newdir"},
	}
	connID := uuid.New()
	rootDir := &explorer.S3Directory{
		ID:   explorer.RootDirID,
		Name: "",
	}
	conn := &connection.Connection{
		BucketName: "test-bucket",
	}

	// Expectations
	connRepo.EXPECT().
		GetSelectedConnection(gomock.Any()).
		Return(conn, nil).
		Times(2)
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(3)
	dirRepo.EXPECT().
		GetByID(gomock.Any(), explorer.RootDirID).
		Return(rootDir, nil).
		Times(2)
	dirRepo.EXPECT().
		GetByID(gomock.Any(), dirID).
		Return(newDir, nil).
		Times(1)

	// When
	err := vm.RefreshDir(dirID)

	// Then
	assert.NoError(t, err)
}

func Test_RefreshDir_ShouldHandleErrorFromDirectoryService(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
	connSvc := mocks_connection.NewMockConnectionService(ctrl)
	connRepo := mocks_connection.NewMockRepository(ctrl)
	dirSvc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		connSvc,
	)
	settingsVm := viewmodel.NewSettingsViewModel(nil)
	vm := viewmodel.NewExplorerViewModel(dirSvc, connRepo, nil, settingsVm)

	dirID := explorer.S3DirectoryID("/test")
	connID := uuid.New()
	rootDir := &explorer.S3Directory{
		ID:   explorer.RootDirID,
		Name: "",
	}
	conn := &connection.Connection{
		BucketName: "test-bucket",
	}

	// Expectations
	connRepo.EXPECT().
		GetSelectedConnection(gomock.Any()).
		Return(conn, nil).
		Times(2)
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(3)
	dirRepo.EXPECT().
		GetByID(gomock.Any(), explorer.RootDirID).
		Return(rootDir, nil).
		Times(2)
	dirRepo.EXPECT().
		GetByID(gomock.Any(), dirID).
		Return(nil, explorer.ErrConnectionNoSet).
		Times(1)

	// When
	err := vm.RefreshDir(dirID)

	// Then
	assert.Error(t, err)
	assert.Equal(t, explorer.ErrConnectionNoSet, err)
}

func Test_RefreshDir_ShouldHandleErrorFromTreeOperations(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
	connSvc := mocks_connection.NewMockConnectionService(ctrl)
	connRepo := mocks_connection.NewMockRepository(ctrl)
	dirSvc := explorer.NewDirectoryService(
		logger,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return dirRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		connSvc,
	)
	settingsVm := viewmodel.NewSettingsViewModel(nil)
	vm := viewmodel.NewExplorerViewModel(dirSvc, connRepo, nil, settingsVm)

	dirID := explorer.S3DirectoryID("/test")
	dir := &explorer.S3Directory{
		ID:   dirID,
		Name: "test",
		Files: []*explorer.S3File{
			{ID: "test/file.txt", Name: "file.txt"},
		},
	}
	connID := uuid.New()
	rootDir := &explorer.S3Directory{
		ID:   explorer.RootDirID,
		Name: "",
	}
	conn := &connection.Connection{
		BucketName: "test-bucket",
	}

	// Expectations
	connRepo.EXPECT().
		GetSelectedConnection(gomock.Any()).
		Return(conn, nil).
		Times(2)
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(3)
	dirRepo.EXPECT().
		GetByID(gomock.Any(), explorer.RootDirID).
		Return(rootDir, nil).
		Times(2)
	dirRepo.EXPECT().
		GetByID(gomock.Any(), dirID).
		Return(dir, nil).
		Times(1)

	// When
	err := vm.RefreshDir(dirID)

	// Then
	assert.NoError(t, err) // Les erreurs d'arbre sont loggées mais ne font pas échouer l'opération
}

