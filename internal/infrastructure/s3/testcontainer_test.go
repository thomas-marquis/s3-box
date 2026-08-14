package s3_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"go.uber.org/mock/gomock"
)

func TestS3EventHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	endpoint, terminate := tu.SetupS3testContainer(ctx, t)
	t.Cleanup(terminate)
	testClient := tu.SetupS3Client(t, endpoint)

	t.Run("test load directory", func(t *testing.T) {
		t.Parallel()

		t.Run("should publish root directory and its content", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
				{Key: "root_file.txt"},
				{Key: "mydir/"},
				{Key: "mydir/file_in_dir.txt"},
				{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
				{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
				{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
				{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
			})
			fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

			rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
			require.NoError(t, err)

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					pl, ok := evt.Payload().(directory.LoadSucceeded)
					res := assert.True(t, ok) &&
						assert.Len(t, pl.SubDirectories, 1) &&
						assert.Equal(t, "/mydir/", pl.SubDirectories[0].Path().String()) &&
						assert.Len(t, pl.Files, 1) &&
						assert.Equal(t, "root_file.txt", pl.Files[0].Name().String())
					close(done)
					return res
				})).
				Times(1)

			eh := s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo)
			defer eh.Destroy()
			eh.Listen()

			// When
			fakeEventChan <- event.New(directory.LoadTriggered{Directory: rootDir})

			// Then
			tu.AssertEventually(t, done)
		})

		t.Run("should returns subdirectory and its content", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
				{Key: "root_file.txt"},
				{Key: "mydir/"},
				{Key: "mydir/file_in_dir.txt"},
				{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
				{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
				{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
				{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
			})
			fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

			dir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "mydir", directory.RootPath)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					pl, ok := evt.Payload().(directory.LoadSucceeded)
					res := assert.True(t, ok) &&
						assert.Len(t, pl.SubDirectories, 2) &&
						assert.Len(t, pl.Files, 1) &&
						assert.Equal(t, "file_in_dir.txt", pl.Files[0].Name().String())
					close(done)
					return res
				})).
				Times(1)

			s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

			// When
			fakeEventChan <- event.New(directory.LoadTriggered{Directory: dir})
			tu.AssertEventually(t, done)
		})

		t.Run("should handle AWS connection without custom endpoint", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
				{Key: "root_file.txt"},
				{Key: "mydir/"},
				{Key: "mydir/file_in_dir.txt"},
				{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
				{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
				{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
				{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
			})

			awsDeck := tu.FakeDeckWithConnections(t,
				tu.FakeAwsConnection(t, bucket))

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, awsDeck, fakeEventChan)

			mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					pl, ok := evt.Payload().(directory.LoadFailed)
					res := assert.True(t, ok) &&
						assert.Error(t, pl.Err) &&
						assert.Contains(t, pl.Err.Error(),
							"InvalidAccessKeyId: The AWS Access Key Id you provided does not exist in our records")
					close(done)
					return res
				})).
				Times(1)

			dir, err := directory.New(tu.FakeAwsConnectionId, directory.RootDirName, nil)
			require.NoError(t, err)

			s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

			// When
			fakeEventChan <- event.New(directory.LoadTriggered{Directory: dir})

			// Then
			tu.AssertEventually(t, done)
		})

		t.Run("should emit a failure event with a UncompletedRename error when markers are detected", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
				{Key: "root_file.txt"},
				{Key: "mydir/"},
				{Key: "mydir/file_in_dir.txt"},
				{Key: "mydir/oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/mydir/newname/"}`)},
				{Key: "mydir/oldname/remaining.txt", Body: strings.NewReader("toto")},
				{Key: "mydir/newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/mydir/oldname/"}`)},
				{Key: "mydir/newname/copied.md", Body: strings.NewReader("lolo")},
			})
			fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

			mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					pl, ok := evt.Payload().(directory.LoadFailed)
					var expErr directory.UncompletedRename
					res := assert.True(t, ok) &&
						assert.ErrorAs(t, pl.Err, &expErr) &&
						assert.Equal(t, directory.Path("/mydir/oldname/"), expErr.SourceDirPath) &&
						assert.Equal(t, directory.Path("/mydir/newname/"), expErr.DestinationDirPath)
					close(done)
					return res
				})).
				Times(1)

			dir := tu.NewNotLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "oldname", "/mydir/")

			s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

			// When
			fakeEventChan <- event.New(directory.LoadTriggered{Directory: dir})
			tu.AssertEventually(t, done)
		})
	})

	t.Run("download file", func(t *testing.T) {
		t.Parallel()

		t.Run("should download file content and publish success", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
				{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("download-me")},
			})
			fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					e, ok := evt.Payload().(directory.DownloadFileSucceeded)
					res := assert.True(t, ok) &&
						assert.Equal(t, "file_in_dir.txt", e.File.Name().String())
					close(done)
					return res
				})).
				Times(1)

			s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

			mydir := tu.NewNotLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "mydir", directory.RootPath)
			file, err := directory.NewFile("file_in_dir.txt", mydir)
			require.NoError(t, err)

			destPath := filepath.Join(t.TempDir(), "file_in_dir.txt")

			// When
			fakeEventChan <- event.New(directory.DownloadFileTriggered{
				ConnectionID: tu.FakeAwsConnectionId,
				DstPath:      destPath,
				File:         file,
			})

			// Then
			tu.AssertEventually(t, done)
			downloaded, err := os.ReadFile(destPath)
			require.NoError(t, err)
			assert.Equal(t, "download-me", string(downloaded))
		})

		t.Run("should publish failure when object is missing", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
				{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("download-me")},
			})
			fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

			mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					e, ok := evt.Payload().(directory.DownloadFileFailed)
					res := assert.True(t, ok) &&
						assert.Error(t, e.Err) &&
						assert.ErrorIs(t, e.Err, directory.ErrNotFound)
					close(done)
					return res
				})).
				Times(1)

			s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

			mydir := tu.NewNotLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "mydir", directory.RootPath)
			file, err := directory.NewFile("missing.txt", mydir)
			require.NoError(t, err)

			destPath := filepath.Join(t.TempDir(), "missing.txt")

			// When & Then
			fakeEventChan <- event.New(directory.DownloadFileTriggered{
				ConnectionID: tu.FakeAwsConnectionId,
				DstPath:      destPath,
				File:         file,
			})
			tu.AssertEventually(t, done)
		})
	})

	t.Run("create file", func(t *testing.T) {
		t.Parallel()

		t.Run("should create an empty file", func(t *testing.T) {
			t.Parallel()
			// Given
			bucket := tu.FakeRandomBucketName()
			tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{})
			fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

			dir := tu.MakeDirectory(t, "mydir",
				tu.WithRootParent(),
				tu.WithConnectionId(tu.FakeAwsConnectionId),
			)
			newFile, err := directory.NewFile("new_file.txt", dir)
			require.NoError(t, err)

			fakeEventChan := make(chan event.Event, 1)
			defer close(fakeEventChan)
			mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

			done := make(chan struct{})
			mockBus.EXPECT().
				Publish(gomock.Cond(func(evt event.Event) bool {
					// Then
					pl, ok := evt.Payload().(directory.CreateFileSucceeded)
					res := assert.True(t, ok) &&
						assert.Equal(t, "new_file.txt", pl.File.Name().String())
					close(done)
					return res
				})).
				Times(1)

			s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

			// When
			fakeEventChan <- event.New(directory.CreateFileTriggered{
				ConnectionID: tu.FakeAwsConnectionId,
				Directory:    dir,
				File:         newFile,
			})
			tu.AssertEventually(t, done)

			tu.AssertObjectContent(t, testClient, bucket, "mydir/new_file.txt", "")
		})
	})
}
