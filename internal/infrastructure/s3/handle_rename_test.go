package s3_test

import (
	"context"
	"fmt"
	http2 "net/http"
	"strings"
	"sync"
	"testing"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"go.uber.org/mock/gomock"
)

func TestNewS3DirectoryRepository_renameFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should rename a file successfully", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "mydir/original.txt", Body: strings.NewReader("original content")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		parentDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "mydir", directory.RootPath)
		originalFile := testutil.AddFileToDirectory(t, parentDir, "original.txt")

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				e, ok := evt.(directory.RenameFileSucceeded)
				res := assert.True(t, ok) &&
					assert.Equal(t, "renamed.txt", e.NewName)
				close(done)
				return res
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameFileTriggered{
			File:      originalFile,
			NewName:   "renamed.txt",
			Directory: parentDir,
		})

		// Then
		testutil.AssertEventually(t, done)

		testutil.AssertObjectNotExists(t, client, bucket, "mydir/original.txt")
		testutil.AssertObjectContent(t, client, bucket, "mydir/renamed.txt", "original content")
	})
}

func TestNewRepositoryImpl_renameDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should ask for user validation before renaming a non-empty directory", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		originalDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		inputEvt := event.New(directory.RenameTriggered{
			Directory: originalDir,
			NewName:   "newname1",
		})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.UserValidationAsked)
				assert.True(t, ok)
				assert.Equal(t, inputEvt, e.Reason)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- inputEvt

		// Then
		testutil.AssertEventually(t, done)

		// Ensure the bucket content is left unchanged until the user has validated the operation
		oldKeys := testutil.ListKeys(t, client, bucket, "originaldir/")
		assert.Len(t, oldKeys, 5)

		testutil.AssertObjectContent(t, client, bucket, "originaldir/file.txt", "file content")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/empty/", "")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/subdir/", "")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/subdir/nested.txt", "nested content")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/subdir/originaldir/more-nested.txt", "more nested content")

		testutil.AssertObjectNotExists(t, client, bucket, "newname1/file.txt")

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "newname1/.s3box-rename-dst")
	})

	t.Run("should rename directory and its content after user had validated it", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		originalDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, originalDir, e.Directory)
				assert.Equal(t, "newname", e.NewName)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		originalEvt := event.New(directory.RenameTriggered{
			Directory: originalDir,
			NewName:   "newname",
		})
		fakeEventChan <- directory.NewUserValidationAcceptedEvent(originalDir, originalEvt)

		// Then
		testutil.AssertEventually(t, done)

		oldKeys := testutil.ListKeys(t, client, bucket, "originaldir/")
		assert.Len(t, oldKeys, 0)

		resKeys := testutil.ListKeys(t, client, bucket, "newname/")
		assert.Len(t, resKeys, 5)

		testutil.AssertObjectContent(t, client, bucket, "newname/file.txt", "file content")
		testutil.AssertObjectContent(t, client, bucket, "newname/empty/", "")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/", "")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/nested.txt", "nested content")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/originaldir/more-nested.txt", "more nested content")

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/file.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/empty/")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/nested.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/originaldir/more-nested.txt")

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-dst")
	})

	t.Run("should rename non-base directory and its content after user had validated it", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		subdir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "subdir", "/originaldir/")
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, subdir, e.Directory)
				assert.Equal(t, "newname", e.NewName)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		originalEvt := event.New(directory.RenameTriggered{
			Directory: originalDir,
			NewName:   "newname",
		})
		fakeEventChan <- directory.NewUserValidationAcceptedEvent(subdir, originalEvt)

		// Then
		testutil.AssertEventually(t, done)

		oldKeys := testutil.ListKeys(t, client, bucket, "originaldir/subdir")
		assert.Len(t, oldKeys, 0)

		resKeys := testutil.ListKeys(t, client, bucket, "originaldir/newname/")
		assert.Len(t, resKeys, 2)

		testutil.AssertObjectContent(t, client, bucket, "originaldir/file.txt", "file content")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/empty/", "")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/newname/", "")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/newname/nested.txt", "nested content")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/newname/originaldir/more-nested.txt", "more nested content")

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/nested.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/originaldir/more-nested.txt")

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/subdir/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/newname/.s3box-rename-dst")
	})

	t.Run("should rename empty directory directly without validation", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(context.TODO(), t, client, bucket, []testutil.FakeS3Object{
			{Key: "base/empty/", Body: strings.NewReader("")},
		})
		dir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "empty", "/base/")
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, dir, e.Directory)
				assert.Equal(t, "newname", e.NewName)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameTriggered{
			Directory: dir,
			NewName:   "newname",
		})

		// Then
		testutil.AssertEventually(t, done)

		testutil.AssertObjectContent(t, client, bucket, "base/newname/", "")
		testutil.AssertObjectNotExists(t, client, bucket, "base/empty/")

		testutil.AssertObjectNotExists(t, client, bucket, "base/empty/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "base/newname/.s3box-rename-dst")
	})

	t.Run("should handle rename failure gracefully and write maker files", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		originalDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				errEvt, ok := evt.(directory.RenameFailed)
				if !assert.True(t, ok) {
					return false
				}
				var expErr directory.UncompletedRename
				return assert.ErrorAs(t, errEvt.Error(), &expErr) &&
					assert.Equal(t, directory.Path("/originaldir/"), expErr.SourceDirPath) &&
					assert.Equal(t, directory.Path("/newname/"), expErr.DestinationDirPath) &&
					assert.Contains(t, errEvt.Error().Error(), "3 error(s) occurred while renaming objects")
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo,
			func(o *awsS3.Options) {
				o.Interceptors.AddBeforeTransmit(&fakeErrorInterceptor{
					CopyErrorForKeys: []string{
						"originaldir/subdir/nested.txt",
						"originaldir/subdir/"},
					DeleteErrorForKeys: []string{
						"originaldir/subdir/originaldir/more-nested.txt"},
				})
			}).Listen()

		fakeEventChan <- directory.NewUserValidationAcceptedEvent(originalDir,
			event.New(directory.RenameTriggered{
				Directory: originalDir,
				NewName:   "newname",
			}))
		testutil.AssertEventually(t, done)

		// copy errors results
		testutil.AssertObjectContent(t, client, bucket, "originaldir/subdir/nested.txt", "nested content")
		testutil.AssertObjectContent(t, client, bucket, "originaldir/subdir/", "")

		testutil.AssertObjectNotExists(t, client, bucket, "newname/subdir/nested.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/subdir/")

		// delete errors results
		testutil.AssertObjectContent(t, client, bucket, "originaldir/subdir/originaldir/more-nested.txt", "more nested content")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/originaldir/more-nested.txt", "more nested content")

		// what's been moved to the dest directory
		testutil.AssertObjectContent(t, client, bucket, "newname/file.txt", "file content")
		testutil.AssertObjectContent(t, client, bucket, "newname/empty/", "")

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/file.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/empty/")

		// check marker files are still there
		testutil.AssertJSONObjectContent(t, client, bucket, "originaldir/.s3box-rename-src", `
		{
			"dstPath": "/newname/"
		}`)
		testutil.AssertJSONObjectContent(t, client, bucket, "newname/.s3box-rename-dst", `
		{
			"srcPath": "/originaldir/"
		}`)
	})

	t.Run("should fails when the destination directory already exists", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "newname/", Body: strings.NewReader("")},
			{Key: "newname/somefile.txt", Body: strings.NewReader("some content")},
		})
		originalDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				errEvt, ok := evt.(directory.RenameFailed)
				if ok {
					assert.Contains(t, errEvt.Error().Error(), "destination directory already exists")
					close(done)
				}
				return ok
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameTriggered{
			Directory: originalDir,
			NewName:   "newname",
		})

		// Then
		testutil.AssertEventually(t, done)

		testutil.AssertObjectNotExists(t, client, bucket, "originaldir/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-dst")
	})

	t.Run("should fails when the src directory already contains a marker file", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/othernewname/"}`)},
		})
		originalDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				errEvt, ok := evt.Payload.(directory.RenameFailed)
				if !assert.True(t, ok) {
					return false
				}
				var expErr directory.UncompletedRename
				return assert.ErrorAs(t, errEvt.Err, &expErr) &&
					assert.Equal(t, directory.Path("/originaldir/"), expErr.SourceDirPath) &&
					assert.Equal(t, directory.Path("/othernewname/"), expErr.DestinationDirPath) &&
					assert.Contains(t, errEvt.Err.Error(), "rename operation has not been completed: /originaldir/ -> /othernewname/")
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameTriggered{
			Directory: originalDir,
			NewName:   "newname",
		})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should rename with default grants when user doesn't have GetObjectACL permission", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(context.TODO(), t, client, bucket, []testutil.FakeS3Object{
			{Key: "base/empty/", Body: strings.NewReader("")},
		})
		dir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "empty", directory.NewPath("base"))
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl, ok := evt.Payload.(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, dir, pl.Directory)
				assert.Equal(t, "newname", pl.NewName)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo, func(opt *awsS3.Options) {
			opt.Interceptors.AddAfterExecution(&fakeGetObjectAclErrorInterceptor{})
		}).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameTriggered{
			Directory: dir,
			NewName:   "newname",
		})

		// Then
		testutil.AssertEventually(t, done)

		testutil.AssertObjectContent(t, client, bucket, "base/newname/", "")
		testutil.AssertObjectNotExists(t, client, bucket, "base/empty/")

		testutil.AssertObjectNotExists(t, client, bucket, "base/empty/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "base/newname/.s3box-rename-dst")
	})
}

func TestRepositoryImpl_resumeRenameDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should successfully resume renaming directory when marker files are present", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "oldname/", Body: strings.NewReader("")},
			{Key: "oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/newname/"}`)},
			{Key: "oldname/file1.txt", Body: strings.NewReader("content 1")},
			{Key: "oldname/file3.txt", Body: strings.NewReader("content 3")},
			{Key: "oldname/subdir/file4.txt", Body: strings.NewReader("content 4")},
			{Key: "oldname/subdir/file6.txt", Body: strings.NewReader("content 6")},

			{Key: "newname/", Body: strings.NewReader("")},
			{Key: "newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/oldname/"}`)},
			{Key: "newname/file1.txt", Body: strings.NewReader("content 1")},
			{Key: "newname/file2.txt", Body: strings.NewReader("content 2")},
			{Key: "newname/subdir/file4.txt", Body: strings.NewReader("content 4")},
			{Key: "newname/subdir/file5.txt", Body: strings.NewReader("content 5")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		oldDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "oldname", directory.RootPath)
		newDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "newname", directory.RootPath)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				e, ok := evt.(directory.RenameSucceeded)
				if ok {
					assert.Equal(t, "newname", e.NewName)
				}
				return ok
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameRecoveryTriggered{
			Directory: oldDir,
			DstDir:    newDir,
			Choice:    directory.RecoveryChoiceRenameResume,
		})

		// Then
		testutil.AssertEventually(t, done)

		// Check everything is moved
		testutil.AssertObjectContent(t, client, bucket, "newname/file1.txt", "content 1")
		testutil.AssertObjectContent(t, client, bucket, "newname/file2.txt", "content 2")
		testutil.AssertObjectContent(t, client, bucket, "newname/file3.txt", "content 3")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/file4.txt", "content 4")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/file5.txt", "content 5")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/file6.txt", "content 6")

		// Check markers are gone
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/file1.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/file2.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/file3.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/subdir/file4.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/subdir/file5.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/subdir/file6.txt")

		testutil.AssertObjectNotExists(t, client, bucket, "oldname/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/.s3box-rename-dst")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-dst")
	})

	t.Run("should successfully rollback renaming directory when marker files are present", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "oldname/", Body: strings.NewReader("")},
			{Key: "oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/newname/"}`)},
			{Key: "oldname/file1.txt", Body: strings.NewReader("content 1")},
			{Key: "oldname/file3.txt", Body: strings.NewReader("content 3")},
			{Key: "oldname/subdir/file4.txt", Body: strings.NewReader("content 4")},
			{Key: "oldname/subdir/file6.txt", Body: strings.NewReader("content 6")},

			{Key: "newname/", Body: strings.NewReader("")},
			{Key: "newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/oldname/"}`)},
			{Key: "newname/file1.txt", Body: strings.NewReader("content 1")},
			{Key: "newname/file2.txt", Body: strings.NewReader("content 2")},
			{Key: "newname/subdir/file4.txt", Body: strings.NewReader("content 4")},
			{Key: "newname/subdir/file5.txt", Body: strings.NewReader("content 5")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		oldDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "oldname", directory.RootPath)
		newDir := testutil.NewLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "newname", directory.RootPath)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				e, ok := evt.(directory.RenameSucceeded)
				if ok {
					assert.Equal(t, "oldname", e.NewName)
				}
				return ok
			})).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameRecoveryTriggered{
			Directory: oldDir,
			DstDir:    newDir,
			Choice:    directory.RecoveryChoiceRenameRollback,
		})

		// Then
		testutil.AssertEventually(t, done)

		// Check everything is moved
		testutil.AssertObjectContent(t, client, bucket, "oldname/file1.txt", "content 1")
		testutil.AssertObjectContent(t, client, bucket, "oldname/file2.txt", "content 2")
		testutil.AssertObjectContent(t, client, bucket, "oldname/file3.txt", "content 3")
		testutil.AssertObjectContent(t, client, bucket, "oldname/subdir/file4.txt", "content 4")
		testutil.AssertObjectContent(t, client, bucket, "oldname/subdir/file5.txt", "content 5")
		testutil.AssertObjectContent(t, client, bucket, "oldname/subdir/file6.txt", "content 6")

		// Check markers are gone
		testutil.AssertObjectNotExists(t, client, bucket, "newname/file1.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/file2.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/file3.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/subdir/file4.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/subdir/file5.txt")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/subdir/file6.txt")

		testutil.AssertObjectNotExists(t, client, bucket, "oldname/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/.s3box-rename-dst")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-dst")
	})

	t.Run("should successfully abort renaming directory when marker files are present", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "oldname/", Body: strings.NewReader("")},
			{Key: "oldname/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/newname/"}`)},
			{Key: "oldname/file1.txt", Body: strings.NewReader("content 1")},
			{Key: "oldname/file3.txt", Body: strings.NewReader("content 3")},
			{Key: "oldname/subdir/file4.txt", Body: strings.NewReader("content 4")},
			{Key: "oldname/subdir/file6.txt", Body: strings.NewReader("content 6")},

			{Key: "newname/", Body: strings.NewReader("")},
			{Key: "newname/.s3box-rename-dst", Body: strings.NewReader(`{"srcPath": "/oldname/"}`)},
			{Key: "newname/file1.txt", Body: strings.NewReader("content 1")},
			{Key: "newname/file2.txt", Body: strings.NewReader("content 2")},
			{Key: "newname/subdir/file4.txt", Body: strings.NewReader("content 4")},
			{Key: "newname/subdir/file5.txt", Body: strings.NewReader("content 5")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		oldDir := testutil.NewNotLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "oldname", directory.RootPath)
		newDir := testutil.NewNotLoadedDirectoryWithConn(t, testutil.FakeAwsConnectionId, "newname", directory.RootPath)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)

		var wg sync.WaitGroup
		wg.Add(2)
		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer wg.Done()
				e, ok := evt.(directory.LoadSucceeded)
				if !ok {
					return ok
				}
				if e.Directory.Name() == "oldname" {
					assert.Len(t, e.Files, 2)
					assert.Len(t, e.SubDirectories, 1)
					assert.Equal(t, "file1.txt", e.Files[0].Name().String())
					assert.Equal(t, "file3.txt", e.Files[1].Name().String())
					assert.Equal(t, "subdir", e.SubDirectories[0].Name())
				} else if e.Directory.Name() == "newname" {
					assert.Len(t, e.Files, 2)
					assert.Len(t, e.SubDirectories, 1)
					assert.Equal(t, "file1.txt", e.Files[0].Name().String())
					assert.Equal(t, "file2.txt", e.Files[1].Name().String())
					assert.Equal(t, "subdir", e.SubDirectories[0].Name())
				} else {
					assert.Fail(t, "unexpected directory")
				}
				return ok
			})).
			Times(2)
		go func() {
			wg.Wait()
			close(done)
		}()

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- event.New(directory.RenameRecoveryTriggered{
			Directory: oldDir,
			DstDir:    newDir,
			Choice:    directory.RecoveryChoiceRenameAbort,
		})

		// Then
		testutil.AssertEventually(t, done)

		// Check everything is moved
		testutil.AssertObjectContent(t, client, bucket, "oldname/file1.txt", "content 1")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/file2.txt")
		testutil.AssertObjectContent(t, client, bucket, "oldname/file3.txt", "content 3")
		testutil.AssertObjectContent(t, client, bucket, "oldname/subdir/file4.txt", "content 4")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/subdir/file5.txt")
		testutil.AssertObjectContent(t, client, bucket, "oldname/subdir/file6.txt", "content 6")

		// Check markers are gone
		testutil.AssertObjectContent(t, client, bucket, "newname/file1.txt", "content 1")
		testutil.AssertObjectContent(t, client, bucket, "newname/file2.txt", "content 2")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/file3.txt")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/file4.txt", "content 4")
		testutil.AssertObjectContent(t, client, bucket, "newname/subdir/file5.txt", "content 5")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/subdir/file6.txt")

		testutil.AssertObjectNotExists(t, client, bucket, "oldname/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "oldname/.s3box-rename-dst")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-src")
		testutil.AssertObjectNotExists(t, client, bucket, "newname/.s3box-rename-dst")
	})
}

type fakeErrorInterceptor struct {
	CopyErrorForKeys   []string
	DeleteErrorForKeys []string
	PutErrorForKeys    []string
}

func (i *fakeErrorInterceptor) BeforeTransmit(ctx context.Context, in *http.InterceptorContext) error {
	if cpyIn, ok := in.Input.(*awsS3.CopyObjectInput); ok {
		for _, key := range i.CopyErrorForKeys {
			if fmt.Sprintf("%s/%s", *cpyIn.Bucket, key) == *cpyIn.CopySource {
				return fmt.Errorf("fake error for key: %s", key)
			}
		}
	}

	if delIn, ok := in.Input.(*awsS3.DeleteObjectInput); ok {
		for _, key := range i.DeleteErrorForKeys {
			if key == *delIn.Key {
				return fmt.Errorf("fake error for key: %s", key)
			}
		}
	}

	if upIn, ok := in.Input.(*awsS3.PutObjectInput); ok {
		for _, key := range i.PutErrorForKeys {
			if key == *upIn.Key {
				return fmt.Errorf("fake error for key: %s", key)
			}
		}
	}

	return nil
}

type fakeGetObjectAclErrorInterceptor struct{}

func (i *fakeGetObjectAclErrorInterceptor) AfterExecution(ctx context.Context, in *http.InterceptorContext) error {
	if _, ok := in.Input.(*awsS3.GetObjectAclInput); ok {
		return &smithy.OperationError{
			ServiceID:     "S3",
			OperationName: "GetObjectAcl",
			Err: &http.ResponseError{
				Response: &http.Response{
					Response: &http2.Response{
						Status:     "403 Forbidden",
						StatusCode: 403,
					},
				},
				Err: &smithy.GenericAPIError{
					Code:    "AccessDenied",
					Message: "api error AccessDenied: User: arn:aws:iam::12345:user/toto is not authorized to perform: s3:GetObjectAcl on resource: \"arn:aws:s3:::mybucket/test/\" because no identity-based policy allows the s3:GetObjectAcl action",
				},
			},
		}
	}
	return nil
}
