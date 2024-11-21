package explorer_test

import (
	"context"
	"errors"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func getDirectoryServiceMocks(t *testing.T) (*gomock.Controller, *mocks_explorer.MockRepository) {
	ctrl := gomock.NewController(t)
	repo := mocks_explorer.NewMockRepository(ctrl)
	return ctrl, repo
}

func Test_Load_ShouldLoadDirectoryContent(t *testing.T) {
	// Given
	_, repo := getDirectoryServiceMocks(t)
	svc := explorer.NewDirectoryService(repo)
	rootDir := explorer.NewDirectory("root", nil)
	currDir := explorer.NewDirectory("mydirectory", rootDir)

	subdir1 := explorer.NewDirectory("dir1", currDir)
	subdir2 := explorer.NewDirectory("dir2", currDir)

	file1 := explorer.NewRemoteFile("file1", currDir)
	file2 := explorer.NewRemoteFile("file2", currDir)
	file3 := explorer.NewRemoteFile("file3", currDir)

	ctx := context.TODO()

	repo.EXPECT().
		ListDirectoryContent(ctx, currDir).
		Return([]*explorer.Directory{subdir1, subdir2}, []*explorer.RemoteFile{file1, file2, file3}, nil).
		Times(1)

	// When
	err := svc.Load(ctx, currDir)

	// Then
	assert.Nil(t, err)
	assert.Equal(t, 2, len(currDir.SubDirectories))
	assert.Equal(t, 3, len(currDir.Files))
	assert.True(t, currDir.IsLoaded)
}

func Test_Load_ShouldReturnErrorWhenFailedToGetContent(t *testing.T) {
	// Given
	_, repo := getDirectoryServiceMocks(t)
	svc := explorer.NewDirectoryService(repo)
	rootDir := explorer.NewDirectory("root", nil)
	currDir := explorer.NewDirectory("mydirectory", rootDir)

	ctx := context.TODO()

	repo.EXPECT().ListDirectoryContent(ctx, currDir).Return(nil, nil, errors.New("ckc"))

	// When
	err := svc.Load(ctx, currDir)

	// Then
	assert.Equal(t, errors.New("impossible to load directory '/root/mydirectory' content: ckc"), err)
}

func Test_Load_ShouldReturnErrConnectionNotSetWhenNoConnection(t *testing.T) {
	// Given
	_, repo := getDirectoryServiceMocks(t)
	svc := explorer.NewDirectoryService(repo)
	rootDir := explorer.NewDirectory("root", nil)
	currDir := explorer.NewDirectory("mydirectory", rootDir)

	ctx := context.TODO()

	repo.EXPECT().ListDirectoryContent(ctx, currDir).Return(nil, nil, explorer.ErrConnectionNoSet)

	// When
	err := svc.Load(ctx, currDir)

	// Then
	assert.Equal(t, explorer.ErrConnectionNoSet, err)
}
