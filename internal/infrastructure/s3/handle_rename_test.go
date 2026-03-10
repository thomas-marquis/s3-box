package s3_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/transport/http"
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

func TestNewS3DirectoryRepository_renameFile(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
		{Key: "mydir/original.txt", Body: strings.NewReader("original content")},
	})

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should rename a file successfully", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

		parentDir := testutil.NewDirectory(t, "mydir", directory.RootPath)
		testutil.AddFileToDirectory(t, parentDir, "original.txt")

		renamedFile, err := directory.NewFile("renamed.txt", parentDir.Path())
		require.NoError(t, err)

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
				e, ok := evt.(directory.FileRenamedSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Equal(t, "renamed.txt", e.File().Name().String())
				close(done)
				return res
			})).
			Times(1)

		_, err = s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		oldName := directory.FileName("original.txt")

		// When
		fakeEventChan <- directory.NewFileRenamedEvent(parentDir, renamedFile, oldName)

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "mydir/original.txt")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "mydir/renamed.txt", "original content")
	})
}

func TestNewS3DirectoryRepository_renameDirectory(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	rootDir := testutil.FakeRootDirectory(t)

	setup := func(baseDirName string) (*directory.Directory, *directory.Directory) {
		setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
			{Key: fmt.Sprintf("%s/", baseDirName), Body: strings.NewReader("")},
			{Key: fmt.Sprintf("%s/file.txt", baseDirName), Body: strings.NewReader("file content")},
			{Key: fmt.Sprintf("%s/empty/", baseDirName), Body: strings.NewReader("")},
			{Key: fmt.Sprintf("%s/subdir/", baseDirName), Body: strings.NewReader("")},
			{Key: fmt.Sprintf("%s/subdir/nested.txt", baseDirName), Body: strings.NewReader("nested content")},
			{Key: fmt.Sprintf("%s/subdir/originaldir/more-nested.txt", baseDirName), Body: strings.NewReader("more nested content")},
		})

		baseDir := testutil.NewDirectory(t, baseDirName, rootDir.Path())
		testutil.AddFileToDirectory(t, baseDir, "file.txt")
		testutil.AddSubDirectoryToDirectory(t, baseDir, "empty")
		subDir := testutil.AddSubDirectoryToDirectory(t, baseDir, "subdir")
		testutil.AddFileToDirectory(t, subDir, "nested.txt")
		subSubDir := testutil.AddSubDirectoryToDirectory(t, subDir, "originaldir")
		testutil.AddFileToDirectory(t, subSubDir, "more-nested.txt")

		return baseDir, subDir
	}

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should ask for user validation before renaming a non-empty directory", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir")

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

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

		inputEvt := directory.NewRenamedEvent(originalDir, "newname1")

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.UserValidationEvent)
				assert.True(t, ok)
				assert.Equal(t, inputEvt, e.Reason())
				close(done)
			}).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- inputEvt

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		// Ensure the bucket content is left unchanged until the user has validated the operation
		oldKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "originaldir/")
		assert.Len(t, oldKeys, 5)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir/file.txt", "file content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir/empty/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir/subdir/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir/subdir/nested.txt", "nested content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir/subdir/originaldir/more-nested.txt", "more nested content")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname1/file.txt")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname1/.s3box-rename-dst")
	})

	t.Run("should rename directory and its content after user had validated it", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir2")

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

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
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenamedSuccessEvent)
				assert.True(t, ok)
				assert.Equal(t, originalDir, e.Directory())
				assert.Equal(t, "newname", e.NewName())
				close(done)
			}).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		originalEvt := directory.NewRenamedEvent(originalDir, "newname")
		fakeEventChan <- directory.NewUserValidationSuccessEvent(originalDir, originalEvt, true)

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		oldKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "originaldir2/")
		assert.Len(t, oldKeys, 0)

		resKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "newname/")
		assert.Len(t, resKeys, 5)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/file.txt", "file content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/empty/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/nested.txt", "nested content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/originaldir/more-nested.txt", "more nested content")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir2/file.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir2/empty/")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir2/subdir/")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir2/subdir/nested.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir2/subdir/originaldir/more-nested.txt")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir2/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname/.s3box-rename-dst")
	})

	t.Run("should rename non-base directory and its content after user had validated it", func(t *testing.T) {
		// Given
		_, subdir := setup("originaldir3")

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

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
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenamedSuccessEvent)
				assert.True(t, ok)
				assert.Equal(t, subdir, e.Directory())
				assert.Equal(t, "newname", e.NewName())
				close(done)
			}).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		originalEvt := directory.NewRenamedEvent(subdir, "newname")
		fakeEventChan <- directory.NewUserValidationSuccessEvent(subdir, originalEvt, true)

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		oldKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "originaldir3/subdir")
		assert.Len(t, oldKeys, 0)

		resKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "originaldir3/newname/")
		assert.Len(t, resKeys, 2)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir3/file.txt", "file content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir3/empty/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir3/newname/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir3/newname/nested.txt", "nested content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir3/newname/originaldir/more-nested.txt", "more nested content")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir3/subdir/")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir3/subdir/nested.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir3/subdir/originaldir/more-nested.txt")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir3/subdir/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir3/newname/.s3box-rename-dst")
	})

	t.Run("should rename empty directory directly without validation", func(t *testing.T) {
		// Given
		dir := testutil.NewDirectory(t, "empty", directory.NewPath("base10"))

		setupS3Bucket(context.TODO(), t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
			{Key: "base10/empty/", Body: strings.NewReader("")},
		})

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

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
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenamedSuccessEvent)
				assert.True(t, ok)
				assert.Equal(t, dir, e.Directory())
				assert.Equal(t, "newname", e.NewName())
				close(done)
			}).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewRenamedEvent(dir, "newname")

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "base10/newname/", "")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "base10/empty/")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "base10/empty/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "base10/newname/.s3box-rename-dst")
	})

	t.Run("should handle rename failure gracefully and write maker files", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir4")

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()
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
				errEvt, ok := evt.(directory.RenamedFailureEvent)
				assert.Equal(t, "3 error(s) occurred while renaming objects", errEvt.Error().Error())
				close(done)
				return ok
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo,
			func(o *awsS3.Options) {
				o.Interceptors.AddBeforeTransmit(&fakeErrorInterceptor{
					CopyErrorForKeys: []string{
						"originaldir4/subdir/nested.txt",
						"originaldir4/subdir/"},
					DeleteErrorForKeys: []string{
						"originaldir4/subdir/originaldir/more-nested.txt"},
				})
			})
		require.NoError(t, err)

		fakeEventChan <- directory.NewUserValidationSuccessEvent(originalDir,
			directory.NewRenamedEvent(originalDir, "newname4"), true)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		// copy errors results
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir4/subdir/nested.txt", "nested content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir4/subdir/", "")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname4/subdir/nested.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname4/subdir/")

		// delete errors results
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir4/subdir/originaldir/more-nested.txt", "more nested content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname4/subdir/originaldir/more-nested.txt", "more nested content")

		// what's been moved to the dest directory
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname4/file.txt", "file content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname4/empty/", "")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir4/file.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir4/empty/")

		// check marker files are still there
		assertJSONObjectContent(t, client, testutil.FakeS3LikeBucketName, "originaldir4/.s3box-rename-src", `
		{
			"dst": "newname4/"
		}`)
		assertJSONObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname4/.s3box-rename-dst", `
		{
			"src": "originaldir4/"
		}`)
	})

	t.Run("should fails when the destination directory already exists", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir_exists")

		// Create the destination directory objects
		putObject(t, client, testutil.FakeS3LikeBucketName, "destination_exists/", strings.NewReader(""))
		putObject(t, client, testutil.FakeS3LikeBucketName, "destination_exists/somefile.txt", strings.NewReader("some content"))

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			AnyTimes()

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				errEvt, ok := evt.(directory.RenamedFailureEvent)
				if ok {
					assert.Contains(t, errEvt.Error().Error(), "destination directory already exists")
					close(done)
				}
				return ok
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewRenamedEvent(originalDir, "destination_exists")

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir_exists/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "destination_exists/.s3box-rename-dst")
	})

	t.Run("should fails when the src directory contains marker file with a different destination", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir_marker_mismatch")

		// Put a marker file that points to a different destination
		putObject(t, client, testutil.FakeS3LikeBucketName, "originaldir_marker_mismatch/.s3box-rename-src", strings.NewReader(`{"dst": "different_dest/"}`))

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			AnyTimes()

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				errEvt, ok := evt.(directory.RenamedFailureEvent)
				if ok {
					assert.Contains(t, errEvt.Error().Error(), "an existing renaming marker is already present")
					close(done)
				}
				return ok
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewRenamedEvent(originalDir, "newname_marker_mismatch")

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir_marker_mismatch/.s3box-rename-src")
	})

	t.Run("should resume renaming when dst is not empty and contains a marker file for the same source", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir_resume")

		// Put a marker file that points to the same destination
		putObject(t, client, testutil.FakeS3LikeBucketName, "originaldir_resume/.s3box-rename-src", strings.NewReader(`{"dst": "newname_resume/"}`))
		putObject(t, client, testutil.FakeS3LikeBucketName, "newname_resume/.s3box-rename-dst", strings.NewReader(`{"src": "originaldir_resume/"}`))

		// Simulate some objects already moved
		putObject(t, client, testutil.FakeS3LikeBucketName, "newname_resume/file.txt", strings.NewReader("file content"))

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0)
		mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			AnyTimes()

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				e, ok := evt.(directory.RenamedSuccessEvent)
				if ok {
					assert.Equal(t, "newname_resume", e.NewName())
					close(done)
				}
				return ok
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		// For resume, we don't need UserValidation because it's a resume of a previously validated operation (or small rename)
		fakeEventChan <- directory.NewRenamedEvent(originalDir, "newname_resume")

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		// Check everything is moved
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname_resume/file.txt", "file content")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir_resume/file.txt")

		// Check markers are gone
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir_resume/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname_resume/.s3box-rename-dst")
	})
}

type fakeErrorInterceptor struct {
	CopyErrorForKeys   []string
	DeleteErrorForKeys []string
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

	return nil
}
