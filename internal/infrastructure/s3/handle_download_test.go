package s3_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	mocks_connection_deck "github.com/thomas-marquis/s3-box/mocks/connection_deck"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

func TestS3DirectoryRepository_downloadFile(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
		{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("download-me")},
	})

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should download file content and publish success", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.ContentDownloadedSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Equal(t, "file_in_dir.txt", e.Content().File().Name().String())
				close(done)
				return res
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("file_in_dir.txt", directory.NewPath("/mydir/"))
		require.NoError(t, err)

		destPath := filepath.Join(t.TempDir(), "file_in_dir.txt")
		content := directory.NewFileContent(file, directory.FromLocalFile(destPath), directory.WithOpenModeWrite())

		// When
		fakeEventChan <- directory.NewContentDownloadedEvent(testutil.FakeS3LikeConnectionId, content)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		downloaded, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, "download-me", string(downloaded))
	})

	t.Run("should publish failure when object is missing", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.ContentDownloadedFailureEvent)
				res := assert.True(t, ok) &&
					assert.Error(t, e.Error()) &&
					assert.ErrorIs(t, e.Error(), directory.ErrNotFound)
				close(done)
				return res
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("missing.txt", directory.NewPath("/mydir/"))
		require.NoError(t, err)

		destPath := filepath.Join(t.TempDir(), "missing.txt")
		content := directory.NewFileContent(file, directory.FromLocalFile(destPath), directory.WithOpenModeWrite())

		// When & Then
		fakeEventChan <- directory.NewContentDownloadedEvent(testutil.FakeS3LikeConnectionId, content)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)
	})
}
