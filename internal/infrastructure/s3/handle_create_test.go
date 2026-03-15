package s3_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"go.uber.org/mock/gomock"
)

func TestNewS3DirectoryRepository_createFile(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	t.Run("should create an empty file", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		setupS3Bucket(ctx, t, client, bucket, []fakeS3Object{})
		fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint, bucket)

		dir := testutil.NewLoadedDirectory(t, "mydir", directory.RootPath)
		newFile, err := directory.NewFile("new_file.txt", dir.Path())
		require.NoError(t, err)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.FileCreatedSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Equal(t, "new_file.txt", e.File().Name().String())
				close(done)
				return res
			})).
			Times(1)

		_, err = s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewFileCreatedEvent(testutil.FakeS3LikeConnectionId, dir, newFile)
		assertEventually(t, done)

		assertObjectContent(t, client, bucket, "mydir/new_file.txt", "")
	})
}
