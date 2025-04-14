package explorer_test

// import (
// 	"context"
// 	"testing"

// 	"github.com/thomas-marquis/s3-box/internal/explorer"
// 	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"

// 	"github.com/gofrs/uuid"
// 	"github.com/stretchr/testify/assert"
// 	"go.uber.org/mock/gomock"
// )

// func getDirectoryServiceMocks(t *testing.T) (*gomock.Controller, *mocks_explorer.MockS3DirectoryRepository) {
// 	ctrl := gomock.NewController(t)
// 	repo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
// 	return ctrl, repo
// }

// func Test_GetRootDirectory_ShouldReturnRootDirectory(t *testing.T) {
// 	// Given
// 	_, repo := getDirectoryServiceMocks(t)
// 	svc := explorer.NewDirectoryService()
// 	connID := uuid.Must(uuid.NewV4())
// 	svc.AddDirectoryRepository(connID, repo)
// 	svc.SetActiveRepository(connID)

// 	expectedDir := explorer.NewS3Directory("root", explorer.RootDirID)
// 	ctx := context.TODO()

// 	repo.EXPECT().
// 		GetByID(ctx, explorer.RootDirID).
// 		Return(expectedDir, nil).
// 		Times(1)

// 	// When
// 	dir, err := svc.GetRootDirectory(ctx)

// 	// Then
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedDir, dir)
// }

// func Test_GetRootDirectory_ShouldReturnErrorWhenNoActiveConnection(t *testing.T) {
// 	// Given
// 	_, repo := getDirectoryServiceMocks(t)
// 	svc := explorer.NewDirectoryService()
// 	connID := uuid.Must(uuid.NewV4())
// 	svc.AddDirectoryRepository(connID, repo)
// 	ctx := context.TODO()

// 	// When
// 	dir, err := svc.GetRootDirectory(ctx)

// 	// Then
// 	assert.Equal(t, explorer.ErrConnectionNoSet, err)
// 	assert.Nil(t, dir)
// }

// func Test_GetDirectoryByID_ShouldReturnDirectory(t *testing.T) {
// 	// Given
// 	_, repo := getDirectoryServiceMocks(t)
// 	svc := explorer.NewDirectoryService()
// 	connID := uuid.Must(uuid.NewV4())
// 	svc.AddDirectoryRepository(connID, repo)
// 	svc.SetActiveRepository(connID)

// 	dirID := explorer.S3DirectoryID("path/to/dir")
// 	expectedDir := explorer.NewS3Directory("dir", explorer.S3DirectoryID("path/to"))
// 	ctx := context.TODO()

// 	repo.EXPECT().
// 		GetByID(ctx, dirID).
// 		Return(expectedDir, nil).
// 		Times(1)

// 	// When
// 	dir, err := svc.GetDirectoryByID(ctx, dirID)

// 	// Then
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedDir, dir)
// }

// func Test_GetDirectoryByID_ShouldReturnErrorWhenNoActiveConnection(t *testing.T) {
// 	// Given
// 	_, repo := getDirectoryServiceMocks(t)
// 	svc := explorer.NewDirectoryService()
// 	connID := uuid.Must(uuid.NewV4())
// 	svc.AddDirectoryRepository(connID, repo)
// 	ctx := context.TODO()

// 	dirID := explorer.S3DirectoryID("path/to/dir")

// 	// When
// 	dir, err := svc.GetDirectoryByID(ctx, dirID)

// 	// Then
// 	assert.Equal(t, explorer.ErrConnectionNoSet, err)
// 	assert.Nil(t, dir)
// }
