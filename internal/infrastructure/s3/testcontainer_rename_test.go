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
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"go.uber.org/mock/gomock"
)

func TestS3EventHandler_rename(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	endpoint, terminate := tu.SetupS3testContainer(ctx, t)
	t.Cleanup(terminate)
	testClient := tu.SetupS3Client(t, endpoint)

	t.Run("should rename a file successfully", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "mydir/original.txt", Body: strings.NewReader("original content")},
		})
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		parentDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "mydir", directory.RootPath)
		originalFile := tu.AddFileToDirectory(t, parentDir, "original.txt")

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				pl, ok := evt.Payload().(directory.RenameFileSucceeded)
				res := assert.True(t, ok) &&
					assert.Equal(t, "renamed.txt", pl.NewName)
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
		tu.AssertEventually(t, done)

		tu.AssertObjectNotExists(t, testClient, bucket, "mydir/original.txt")
		tu.AssertObjectContent(t, testClient, bucket, "mydir/renamed.txt", "original content")
	})

	t.Run("should ask for user validation before renaming a non-empty directory", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		originalDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

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
				pl, ok := evt.Payload().(directory.UserValidationAsked)
				assert.True(t, ok)
				assert.Equal(t, inputEvt, pl.Reason)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		fakeEventChan <- inputEvt

		// Then
		tu.AssertEventually(t, done)

		// Ensure the bucket content is left unchanged until the user has validated the operation
		oldKeys := tu.ListKeys(t, testClient, bucket, "originaldir/")
		assert.Len(t, oldKeys, 5)

		var wg sync.WaitGroup
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/file.txt", "file content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/empty/", "", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/subdir/", "", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/subdir/nested.txt", "nested content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/subdir/originaldir/more-nested.txt", "more nested content", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname1/file.txt", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname1/.s3box-rename-dst", &wg)
		wg.Wait()
	})

	t.Run("should rename directory and its content after user had validated it", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		originalDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl, ok := evt.Payload().(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, originalDir, pl.Directory)
				assert.Equal(t, "newname", pl.NewName)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		originalEvt := event.New(directory.RenameTriggered{
			Directory: originalDir,
			NewName:   "newname",
		})
		fakeEventChan <- event.New(directory.UserValidationAccepted{
			Directory: originalDir,
			Reason:    originalEvt,
		})

		// Then
		tu.AssertEventually(t, done)

		oldKeys := tu.ListKeys(t, testClient, bucket, "originaldir/")
		assert.Len(t, oldKeys, 0)

		resKeys := tu.ListKeys(t, testClient, bucket, "newname/")
		assert.Len(t, resKeys, 5)

		var wg sync.WaitGroup
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file.txt", "file content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/empty/", "", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/", "", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/nested.txt", "nested content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/originaldir/more-nested.txt", "more nested content", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/file.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/empty/", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/nested.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/originaldir/more-nested.txt", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-dst", &wg)
		wg.Wait()
	})

	t.Run("should rename non-base directory and its content after user had validated it", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		subdir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "subdir", "/originaldir/")
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl, ok := evt.Payload().(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, subdir, pl.Directory)
				assert.Equal(t, "newname", pl.NewName)
				close(done)
			}).
			Times(1)

		s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo).Listen()

		// When
		originalEvt := event.New(directory.RenameTriggered{
			Directory: subdir,
			NewName:   "newname",
		})
		fakeEventChan <- event.New(directory.UserValidationAccepted{
			Directory: subdir,
			Reason:    originalEvt,
		})

		// Then
		tu.AssertEventually(t, done)

		var wg sync.WaitGroup

		wg.Go(func() {
			oldKeys := tu.ListKeys(t, testClient, bucket, "originaldir/subdir")
			assert.Len(t, oldKeys, 0)
		})

		wg.Go(func() {
			resKeys := tu.ListKeys(t, testClient, bucket, "originaldir/newname/")
			assert.Len(t, resKeys, 2)
		})

		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/file.txt", "file content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/empty/", "", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/newname/", "", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/newname/nested.txt", "nested content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/newname/originaldir/more-nested.txt", "more nested content", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/nested.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/originaldir/more-nested.txt", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/subdir/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/newname/.s3box-rename-dst", &wg)
		wg.Wait()
	})

	t.Run("should rename empty directory directly without validation", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(context.TODO(), t, testClient, bucket, []tu.FakeS3Object{
			{Key: "base/empty/", Body: strings.NewReader("")},
		})
		dir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "empty", "/base/")
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl, ok := evt.Payload().(directory.RenameSucceeded)
				assert.True(t, ok)
				assert.Equal(t, dir, pl.Directory)
				assert.Equal(t, "newname", pl.NewName)
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
		tu.AssertEventually(t, done)

		var wg sync.WaitGroup
		tu.AssertObjectContentAsync(t, testClient, bucket, "base/newname/", "", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "base/empty/", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "base/empty/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "base/newname/.s3box-rename-dst", &wg)
		wg.Wait()
	})

	t.Run("should handle rename failure gracefully and write marker files", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/empty/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/", Body: strings.NewReader("")},
			{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
			{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
		})
		originalDir := tu.MakeDirectory(t, "originaldir",
			tu.WithRootParent(),
			tu.WithConnectionId(tu.FakeAwsConnectionId))
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				errPl, ok := evt.Payload().(directory.RenameFailed)
				if !assert.True(t, ok) {
					return false
				}
				var expErr directory.UncompletedRename
				return assert.ErrorAs(t, errPl.Err, &expErr) &&
					assert.Equal(t, directory.Path("/originaldir/"), expErr.SourceDirPath) &&
					assert.Equal(t, directory.Path("/newname/"), expErr.DestinationDirPath) &&
					assert.Contains(t, errPl.Err.Error(), "3 error(s) occurred while renaming objects")
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

		fakeEventChan <- event.New(directory.UserValidationAccepted{
			Directory: originalDir,
			Reason: event.New(directory.RenameTriggered{
				Directory: originalDir,
				NewName:   "newname",
			}),
		})
		tu.AssertEventually(t, done)

		var wg sync.WaitGroup

		// copy errors results
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/subdir/nested.txt", "nested content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/subdir/", "", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/subdir/nested.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/subdir/", &wg)

		// delete errors results
		tu.AssertObjectContentAsync(t, testClient, bucket, "originaldir/subdir/originaldir/more-nested.txt", "more nested content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/originaldir/more-nested.txt", "more nested content", &wg)

		// what's been moved to the dest directory
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file.txt", "file content", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/empty/", "", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/file.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "originaldir/empty/", &wg)

		// check marker files are still there
		tu.AssertJSONObjectContentAsync(t, testClient, bucket, "originaldir/.s3box-rename-src", `
		{
			"dstPath": "/newname/"
		}`, &wg)
		tu.AssertJSONObjectContentAsync(t, testClient, bucket, "newname/.s3box-rename-dst", `
		{
			"srcPath": "/originaldir/"
		}`, &wg)
		wg.Wait()
	})

	t.Run("should fails when the destination directory already exists", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "newname/", Body: strings.NewReader("")},
			{Key: "newname/somefile.txt", Body: strings.NewReader("some content")},
		})
		originalDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				errPl, ok := evt.Payload().(directory.RenameFailed)
				if ok {
					assert.Contains(t, errPl.Err.Error(), "destination directory already exists")
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
		tu.AssertEventually(t, done)

		tu.AssertObjectNotExists(t, testClient, bucket, "originaldir/.s3box-rename-src")
		tu.AssertObjectNotExists(t, testClient, bucket, "newname/.s3box-rename-dst")
	})

	t.Run("should fails when the src directory already contains a marker file", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
			{Key: "originaldir/", Body: strings.NewReader("")},
			{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
			{Key: "originaldir/.s3box-rename-src", Body: strings.NewReader(`{"dstPath": "/othernewname/"}`)},
		})
		originalDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "originaldir", directory.RootPath)
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				errEvt, ok := evt.Payload().(directory.RenameFailed)
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
		tu.AssertEventually(t, done)
	})

	t.Run("should rename with default grants when user doesn't have GetObjectACL permission", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(context.TODO(), t, testClient, bucket, []tu.FakeS3Object{
			{Key: "base/empty/", Body: strings.NewReader("")},
		})
		dir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "empty", directory.NewPath("base"))
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl, ok := evt.Payload().(directory.RenameSucceeded)
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
		tu.AssertEventually(t, done)

		tu.AssertObjectContent(t, testClient, bucket, "base/newname/", "")
		tu.AssertObjectNotExists(t, testClient, bucket, "base/empty/")

		tu.AssertObjectNotExists(t, testClient, bucket, "base/empty/.s3box-rename-src")
		tu.AssertObjectNotExists(t, testClient, bucket, "base/newname/.s3box-rename-dst")
	})

	t.Run("should successfully resume renaming directory when marker files are present", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
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
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		oldDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "oldname", directory.RootPath)
		newDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "newname", directory.RootPath)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				pl, ok := evt.Payload().(directory.RenameSucceeded)
				if ok {
					assert.Equal(t, "newname", pl.NewName)
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
		tu.AssertEventually(t, done)

		var wg sync.WaitGroup

		// Check everything is moved
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file1.txt", "content 1", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file2.txt", "content 2", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file3.txt", "content 3", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/file4.txt", "content 4", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/file5.txt", "content 5", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/file6.txt", "content 6", &wg)

		// Check markers are gone
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/file1.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/file2.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/file3.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/subdir/file4.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/subdir/file5.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/subdir/file6.txt", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/.s3box-rename-dst", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-dst", &wg)
		wg.Wait()
	})

	t.Run("should successfully rollback renaming directory when marker files are present", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
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
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		oldDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "oldname", directory.RootPath)
		newDir := tu.NewLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "newname", directory.RootPath)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer close(done)
				pl, ok := evt.Payload().(directory.RenameSucceeded)
				if ok {
					assert.Equal(t, "oldname", pl.NewName)
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
		tu.AssertEventually(t, done)

		var wg sync.WaitGroup

		// Check everything is moved
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/file1.txt", "content 1", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/file2.txt", "content 2", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/file3.txt", "content 3", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/subdir/file4.txt", "content 4", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/subdir/file5.txt", "content 5", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/subdir/file6.txt", "content 6", &wg)

		// Check markers are gone
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/file1.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/file2.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/file3.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/subdir/file4.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/subdir/file5.txt", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/subdir/file6.txt", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/.s3box-rename-dst", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-dst", &wg)
		wg.Wait()
	})

	t.Run("should successfully abort renaming directory when marker files are present", func(t *testing.T) {
		t.Parallel()
		// Given
		bucket := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
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
		fakeDeck := tu.FakeDeckWithAwsConnection(t, endpoint, bucket)

		oldDir := tu.NewNotLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "oldname", directory.RootPath)
		newDir := tu.NewNotLoadedDirectoryWithConn(t, tu.FakeAwsConnectionId, "newname", directory.RootPath)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)

		var wg1 sync.WaitGroup
		wg1.Add(2)
		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				defer wg1.Done()
				pl, ok := evt.Payload().(directory.LoadSucceeded)
				if !ok {
					return ok
				}
				if pl.Directory.Name() == "oldname" {
					assert.Len(t, pl.Files, 2)
					assert.Len(t, pl.SubDirectories, 1)
					assert.Equal(t, "file1.txt", pl.Files[0].Name().String())
					assert.Equal(t, "file3.txt", pl.Files[1].Name().String())
					assert.Equal(t, "subdir", pl.SubDirectories[0].Name())
				} else if pl.Directory.Name() == "newname" {
					assert.Len(t, pl.Files, 2)
					assert.Len(t, pl.SubDirectories, 1)
					assert.Equal(t, "file1.txt", pl.Files[0].Name().String())
					assert.Equal(t, "file2.txt", pl.Files[1].Name().String())
					assert.Equal(t, "subdir", pl.SubDirectories[0].Name())
				} else {
					assert.Fail(t, "unexpected directory")
				}
				return ok
			})).
			Times(2)
		go func() {
			wg1.Wait()
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
		tu.AssertEventually(t, done)

		var wg sync.WaitGroup

		// Check everything is moved
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/file1.txt", "content 1", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/file2.txt", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/file3.txt", "content 3", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/subdir/file4.txt", "content 4", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/subdir/file5.txt", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "oldname/subdir/file6.txt", "content 6", &wg)

		// Check markers are gone
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file1.txt", "content 1", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/file2.txt", "content 2", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/file3.txt", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/file4.txt", "content 4", &wg)
		tu.AssertObjectContentAsync(t, testClient, bucket, "newname/subdir/file5.txt", "content 5", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/subdir/file6.txt", &wg)

		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "oldname/.s3box-rename-dst", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-src", &wg)
		tu.AssertObjectNotExistsAsync(t, testClient, bucket, "newname/.s3box-rename-dst", &wg)

		wg.Wait()
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
			if s3client.WeiredEscape(*cpyIn.Bucket, key) == *cpyIn.CopySource {
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
