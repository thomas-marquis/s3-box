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
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should create an empty file", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		dir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "mydir", directory.RootPath)
		newFile, err := directory.NewFile("new_file.txt", dir)
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
					assert.Equal(t, "new_file.txt", e.File.Name().String())
				close(done)
				return res
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- directory.NewFileCreatedEvent(testutil.FakeAwsConnectionId, dir, newFile)
		testutil.AssertEventually(t, done)

		testutil.AssertObjectContent(t, client, bucket, "mydir/new_file.txt", "")
	})
}
