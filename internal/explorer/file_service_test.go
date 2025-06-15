package explorer_test

import (
	"context"
	"testing"

	"github.com/thomas-marquis/s3-box/internal/explorer"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connections"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func getFileServiceMocks(t *testing.T) (*gomock.Controller, *mocks_explorer.MockS3FileRepository, *mocks_connection.MockConnectionService) {
	ctrl := gomock.NewController(t)
	repo := mocks_explorer.NewMockS3FileRepository(ctrl)
	connSvc := mocks_connection.NewMockConnectionService(ctrl)
	return ctrl, repo, connSvc
}

func Test_GetContent_ShouldReturnContent(t *testing.T) {
	// Given
	ctrl, repo, connSvc := getFileServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewFileService(logger, func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
		return repo, nil
	}, connSvc)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(1)

	file := &explorer.S3File{
		ID:          "test/file.txt",
		DirectoryID: "test",
		Name:        "file.txt",
	}
	ctx := context.TODO()
	expectedContent := []byte("test content")

	repo.EXPECT().
		GetContent(ctx, file.ID).
		Return(expectedContent, nil).
		Times(1)

	// When
	content, err := svc.GetContent(ctx, file)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

func Test_GetContent_ShouldReturnErrorWhenNoActiveConnection(t *testing.T) {
	// Given
	ctrl, repo, connSvc := getFileServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewFileService(logger, func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
		return repo, nil
	}, connSvc)

	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(uuid.Nil, explorer.ErrConnectionNoSet).
		Times(1)

	file := &explorer.S3File{
		ID:          "test/file.txt",
		DirectoryID: "test",
		Name:        "file.txt",
	}
	ctx := context.TODO()

	// When
	content, err := svc.GetContent(ctx, file)

	// Then
	assert.Equal(t, explorer.ErrConnectionNoSet, err)
	assert.Nil(t, content)
}

func Test_GetContent_ShouldReturnErrorWhenRepositoryError(t *testing.T) {
	// Given
	ctrl, repo, connSvc := getFileServiceMocks(t)
	defer ctrl.Finish()

	logger := zap.NewNop()
	svc := explorer.NewFileService(logger, func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
		return repo, nil
	}, connSvc)

	connID := uuid.New()
	connSvc.EXPECT().
		GetActiveConnectionID(gomock.Any()).
		Return(connID, nil).
		Times(1)

	file := &explorer.S3File{
		ID:          "test/file.txt",
		DirectoryID: "test",
		Name:        "file.txt",
	}
	ctx := context.TODO()
	expectedErr := assert.AnError

	repo.EXPECT().
		GetContent(ctx, file.ID).
		Return(nil, expectedErr).
		Times(1)

	// When
	content, err := svc.GetContent(ctx, file)

	// Then
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, content)
}

