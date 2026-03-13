package s3_test

import (
	"context"
	"fmt"
	http2 "net/http"
	"strings"
	"testing"
	"time"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
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

func TestNewRepositoryImpl_renameDirectory(t *testing.T) {
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

		inputEvt := directory.NewRenameEvent(originalDir, "newname1")

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
		originalEvt := directory.NewRenameEvent(originalDir, "newname")
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
		originalEvt := directory.NewRenameEvent(subdir, "newname")
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
		fakeEventChan <- directory.NewRenameEvent(dir, "newname")

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
				defer close(done)
				errEvt, ok := evt.(directory.RenameFailureEvent)
				if !assert.True(t, ok) {
					return false
				}
				var expErr directory.UncompletedRename
				return assert.ErrorAs(t, errEvt.Error(), &expErr) &&
					assert.Equal(t, directory.Path("/originaldir4/"), expErr.SourceDirPath) &&
					assert.Equal(t, directory.Path("/newname4/"), expErr.DestinationDirPath) &&
					assert.Contains(t, errEvt.Error().Error(), "3 error(s) occurred while renaming objects")
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
			directory.NewRenameEvent(originalDir, "newname4"), true)
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
			"dstPath": "/newname4/"
		}`)
		assertJSONObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname4/.s3box-rename-dst", `
		{
			"srcPath": "/originaldir4/"
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
				errEvt, ok := evt.(directory.RenameFailureEvent)
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
		fakeEventChan <- directory.NewRenameEvent(originalDir, "destination_exists")

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

	t.Run("should fails when the src directory already contains a marker file", func(t *testing.T) {
		// Given
		originalDir, _ := setup("originaldir_marker_mismatch")

		putObject(t, client, testutil.FakeS3LikeBucketName, "originaldir_marker_mismatch/.s3box-rename-src",
			strings.NewReader(`{"dstPath": "/different_dest/"}`))

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
				defer close(done)
				errEvt, ok := evt.(directory.RenameFailureEvent)
				if !assert.True(t, ok) {
					return false
				}
				var expErr directory.UncompletedRename
				return assert.ErrorAs(t, errEvt.Error(), &expErr) &&
					assert.Equal(t, directory.Path("/originaldir_marker_mismatch/"), expErr.SourceDirPath) &&
					assert.Equal(t, directory.Path("/different_dest/"), expErr.DestinationDirPath) &&
					assert.Contains(t, errEvt.Error().Error(), "rename operation has not been completed: /originaldir_marker_mismatch/ -> /different_dest/")
			})).
			Times(1)

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewRenameEvent(originalDir, "newname_marker_mismatch")

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("should rename with default grants when user doesn't have GetObjectACL permission", func(t *testing.T) {
		// Given
		dir := testutil.NewDirectory(t, "empty", directory.NewPath("base11"))

		setupS3Bucket(context.TODO(), t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
			{Key: "base11/empty/", Body: strings.NewReader("")},
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

		_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo, func(opt *awsS3.Options) {
			opt.Interceptors.AddAfterExecution(&fakeGetObjectAclErrorInterceptor{})
		})
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewRenameEvent(dir, "newname")

		// Then
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "base11/newname/", "")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "base11/empty/")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "base11/empty/.s3box-rename-src")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "base11/newname/.s3box-rename-dst")
	})
}

func TestRepositoryImpl_resumeRenameDirectory(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	oldDir := testutil.NewDirectory(t, "oldname", directory.RootPath)
	newDir := testutil.NewDirectory(t, "newname", directory.RootPath)
	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	for _, evt := range []directory.RenameResumeEvent{
		directory.NewRenameResumeEvent(oldDir, true, "/newname/"),
		directory.NewRenameResumeEvent(newDir, false, "/oldname/"),
	} {
		t.Run(fmt.Sprintf("should successfully resume renaming from %s directory when marker files are present", evt.Directory().Name()), func(t *testing.T) {
			// Given
			setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
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
						assert.Equal(t, "newname", e.NewName())
						close(done)
					}
					return ok
				})).
				Times(1)

			_, err := s3.NewRepositoryImpl(mockConnRepo, mockBus, mockNotifRepo)
			require.NoError(t, err)

			// When
			fakeEventChan <- directory.NewRenameResumeEvent(oldDir, true, "/newname/")

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
			assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/file1.txt", "content 1")
			assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/file2.txt", "content 2")
			assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/file3.txt", "content 3")
			assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/file4.txt", "content 4")
			assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/file5.txt", "content 5")
			assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/file6.txt", "content 6")

			// Check markers are gone
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/file1.txt")
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/file2.txt")
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/file3.txt")
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/subdir/file4.txt")
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/subdir/file5.txt")
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/subdir/file6.txt")

			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "oldname/.s3box-rename-src")
			assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname/.s3box-rename-dst")
		})
	}
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
