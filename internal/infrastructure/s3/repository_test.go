package s3_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestNewS3DirectoryRepository_GetFileContent(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	bucket := testutil.FakeRandomBucketName()
	testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
		{Key: "root_file.txt", Body: strings.NewReader("coucou")},
		{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("lolo")},
	})
	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint, bucket)

	t.Run("should return the file content", func(t *testing.T) {
		// Given

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		repo, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		rootDir := testutil.FakeNotLoadedRootDirectory(t)
		file, err := directory.NewFile("root_file.txt", rootDir)
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), testutil.FakeS3LikeConnectionId, file)

		// Then
		assert.NoError(t, err)

		f, err := res.Open()
		assert.NoError(t, err)
		var resContent []byte
		resContent, err = io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "coucou", string(resContent))
	})

	t.Run("should return the file content", func(t *testing.T) {
		// Given
		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		repo, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		mydir := testutil.NewNotLoadedDirectory(t, "mydir", directory.RootPath)
		file, err := directory.NewFile("file_in_dir.txt", mydir)
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), testutil.FakeS3LikeConnectionId, file)

		// Then
		assert.NoError(t, err)

		f, err := res.Open()
		assert.NoError(t, err)
		var resContent []byte
		resContent, err = io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "lolo", string(resContent))
	})
}
