package directory_test

// import (
// 	"testing"
//
// 	"github.com/thomas-marquis/s3-box/internal/domain/directory"
// 	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"
// 	"go.uber.org/mock/gomock"
// )
//
// func getDirectoryServiceMocks(t *testing.T) (
// 	*gomock.Controller,
// 	*mocks_explorer.MockS3DirectoryRepository,
// 	*mocks_explorer.MockS3FileRepository,
// ) {
// 	ctrl := gomock.NewController(t)
// 	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
// 	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
// 	return ctrl, dirRepo, fileRepo
// }
//
// func Test_GetRoot_ShouldReturnRootDirectory(t *testing.T) {
// 	// Given
// 	ctrl, dirRepo, _ := getDirectoryServiceMocks(t)
// 	defer ctrl.Finish()
//
// 	svc := directory.NewService()
//
// 	connID := uuid.New()
// 	connSvc.EXPECT().
// 		GetActiveConnectionID(gomock.Any()).
// 		Return(connID, nil).
// 		Times(1)
//
// 	expectedDir := &explorer.S3Directory{
// 		ID:   explorer.RootDirID,
// 		Name: "",
// 	}
// 	ctx := context.TODO()
//
// 	dirRepo.EXPECT().
// 		GetByID(ctx, explorer.RootDirID).
// 		Return(expectedDir, nil).
// 		Times(1)
//
// 	// When
// 	dir, err := svc.GetRootDirectory(ctx)
//
// 	// Then
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedDir, dir)
// }
//
// func Test_GetRootDirectory_ShouldReturnErrorWhenNoActiveConnection(t *testing.T) {
// 	// Given
// 	ctrl, dirRepo, _, connSvc := getDirectoryServiceMocks(t)
// 	defer ctrl.Finish()
//
// 	logger := zap.NewNop()
// 	svc := explorer.NewDirectoryService(
// 		logger,
// 		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
// 			return dirRepo, nil
// 		},
// 		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
// 			return nil, nil
// 		},
// 		connSvc,
// 	)
//
// 	connSvc.EXPECT().
// 		GetActiveConnectionID(gomock.Any()).
// 		Return(uuid.Nil, explorer.ErrConnectionNoSet).
// 		Times(1)
//
// 	ctx := context.TODO()
//
// 	// When
// 	dir, err := svc.GetRootDirectory(ctx)
//
// 	// Then
// 	assert.Equal(t, explorer.ErrConnectionNoSet, err)
// 	assert.Nil(t, dir)
// }
