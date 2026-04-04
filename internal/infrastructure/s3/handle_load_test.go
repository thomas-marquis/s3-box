package s3_test

import (
	"context"
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

func TestS3DirectoryRepository_loadDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should publish root directory and its content", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "root_file.txt"},
			{Key: "mydir/"},
			{Key: "mydir/file_in_dir.txt"},
			{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
			{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
			{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
			{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.LoadSucceeded)
				res := assert.True(t, ok) &&
					assert.Len(t, e.SubDirectories, 1) &&
					assert.Equal(t, "/mydir/", e.SubDirectories[0].Path().String()) &&
					assert.Len(t, e.Files, 1) &&
					assert.Equal(t, "root_file.txt", e.Files[0].Name().String())
				close(done)
				return res
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.LoadTriggered{Directory: rootDir})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should returns subdirectory and its content", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "root_file.txt"},
			{Key: "mydir/"},
			{Key: "mydir/file_in_dir.txt"},
			{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
			{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
			{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
			{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		dir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "mydir", directory.RootPath)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.LoadSucceeded)
				res := assert.True(t, ok) &&
					assert.Len(t, e.SubDirectories, 2) &&
					assert.Len(t, e.Files, 1) &&
					assert.Equal(t, "file_in_dir.txt", e.Files[0].Name().String())
				close(done)
				return res
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.LoadTriggered{Directory: dir})
		testutil.AssertEventually(t, done)
	})

	t.Run("should handle AWS connection without custom endpoint", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "root_file.txt"},
			{Key: "mydir/"},
			{Key: "mydir/file_in_dir.txt"},
			{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
			{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
			{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
			{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
		})

		awsDeck := testutil.FakeDeckWithConnections(t,
			testutil.FakeAwsConnection(t, bucket))

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, awsDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				// Then
				e, ok := evt.(directory.LoadFailed)
				res := assert.True(t, ok) &&
					assert.Error(t, e.Error()) &&
					assert.Contains(t, e.Error().Error(),
						"InvalidAccessKeyId: The AWS Access Key Id you provided does not exist in our records")
				close(done)
				return res
			})).
			Times(1)

		dir, err := directory.New(testutil.FakeAwsConnectionId, directory.RootDirName, nil)
		require.NoError(t, err)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.LoadTriggered{Directory: dir})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should emit a failure event with a UncompletedRename error when markers are detected", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "root_file.txt"},
			{Key: "mydir/"},
			{Key: "mydir/file_in_dir.txt"},
			{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
			{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
			{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
			{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
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
				e, ok := evt.(directory.LoadFailed)
				var expErr directory.UncompletedRename
				res := assert.True(t, ok) &&
					assert.ErrorAs(t, e.Error(), &expErr) &&
					assert.Equal(t, directory.Path("/mydir/oldname/"), expErr.SourceDirPath) &&
					assert.Equal(t, directory.Path("/mydir/newname/"), expErr.DestinationDirPath)
				close(done)
				return res
			})).
			Times(1)

		dir := testutil.NewNotLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "oldname", "/mydir/")

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.LoadTriggered{Directory: dir})
		testutil.AssertEventually(t, done)
	})
}
