package s3_test

import (
	"context"
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

func TestS3DirectoryRepository_loadDirectory(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
		{Key: "root_file.txt"},
		{Key: "mydir/"},
		{Key: "mydir/file_in_dir.txt"},
	})

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should publish root directory and its content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		rootDir := testutil.FakeRootDirectory(t)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.LoadSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Len(t, e.SubDirectories(), 1) &&
					assert.Equal(t, "/mydir/", e.SubDirectories()[0].Path().String()) &&
					assert.Len(t, e.Files(), 1) &&
					assert.Equal(t, "root_file.txt", e.Files()[0].Name().String())
				close(done)
				return res
			})).
			Times(1)

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewLoadEvent(rootDir)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("should returns subdirectory and its content", func(t *testing.T) {
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

		dir := testutil.NewDirectory(t, "mydir", directory.RootPath)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.LoadSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Len(t, e.SubDirectories(), 0) &&
					assert.Len(t, e.Files(), 1) &&
					assert.Equal(t, "file_in_dir.txt", e.Files()[0].Name().String())
				close(done)
				return res
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewLoadEvent(dir)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("should handle AWS connection without custom endpoint", func(t *testing.T) {
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
			Return(event.NewSubscriber(fakeEventChan)).
			AnyTimes()

		awsDeck := testutil.FakeDeckWithConnections(t,
			testutil.FakeAwsConnection(t))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(awsDeck, nil).
			Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.LoadFailureEvent)
				res := assert.True(t, ok) &&
					assert.Error(t, e.Error()) &&
					assert.Contains(t, e.Error().Error(),
						"InvalidAccessKeyId: The AWS Access Key Id you provided does not exist in our records")
				close(done)
				return res
			})).
			Times(1)

		dir, err := directory.New(testutil.FakeAwsConnectionId, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		_, err = s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewLoadEvent(dir)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 15*time.Second, 100*time.Millisecond)
	})
}
