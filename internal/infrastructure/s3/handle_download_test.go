package s3_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"go.uber.org/mock/gomock"
)

func TestS3DirectoryRepository_downloadFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should download file content and publish success", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("download-me")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.FileDownloadSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Equal(t, "file_in_dir.txt", e.File.Name().String())
				close(done)
				return res
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		mydir := testutil.NewNotLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "mydir", directory.RootPath)
		file, err := directory.NewFile("file_in_dir.txt", mydir)
		require.NoError(t, err)

		destPath := filepath.Join(t.TempDir(), "file_in_dir.txt")

		// When
		fakeEventChan <- directory.NewFileDownloadEvent(testutil.FakeAwsConnectionId, file, destPath)

		// Then
		testutil.AssertEventually(t, done)
		downloaded, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, "download-me", string(downloaded))
	})

	t.Run("should publish failure when object is missing", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("download-me")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.FileDownloadFailureEvent)
				res := assert.True(t, ok) &&
					assert.Error(t, e.Error()) &&
					assert.ErrorIs(t, e.Error(), directory.ErrNotFound)
				close(done)
				return res
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		mydir := testutil.NewNotLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "mydir", directory.RootPath)
		file, err := directory.NewFile("missing.txt", mydir)
		require.NoError(t, err)

		destPath := filepath.Join(t.TempDir(), "missing.txt")

		// When & Then
		fakeEventChan <- directory.NewFileDownloadEvent(testutil.FakeAwsConnectionId, file, destPath)
		testutil.AssertEventually(t, done)
	})
}
