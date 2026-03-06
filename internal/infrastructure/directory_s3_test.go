package infrastructure_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
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

		_, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

		_, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

		_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

func TestNewS3DirectoryRepository_GetFileContent(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
		{Key: "root_file.txt", Body: strings.NewReader("coucou")},
		{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("lolo")},
	})

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should return the file content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("root_file.txt", directory.RootPath)
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), testutil.FakeS3LikeConnectionId, file)

		// Then
		assert.NoError(t, err)

		f, err := res.Open()
		assert.NoError(t, err)
		var resContent []byte
		resContent, err = io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "coucou", string(resContent))
	})

	t.Run("should return the file content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("file_in_dir.txt", directory.NewPath("/mydir/"))
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), testutil.FakeS3LikeConnectionId, file)

		// Then
		assert.NoError(t, err)

		f, err := res.Open()
		assert.NoError(t, err)
		var resContent []byte
		resContent, err = io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "lolo", string(resContent))
	})
}

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

		_, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

		_, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

func TestNewS3DirectoryRepository_createFile(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{})

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should create an empty file", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		dir := testutil.NewDirectory(t, "mydir", directory.RootPath)
		newFile, err := directory.NewFile("new_file.txt", dir.Path())
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
				// Then
				e, ok := evt.(directory.FileCreatedSuccessEvent)
				res := assert.True(t, ok) &&
					assert.Equal(t, "new_file.txt", e.File().Name().String())
				close(done)
				return res
			})).
			Times(1)

		_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		fakeEventChan <- directory.NewFileCreatedEvent(testutil.FakeS3LikeConnectionId, dir, newFile)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "mydir/new_file.txt", "")
	})
}

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

		_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
		{Key: "originaldir/", Body: strings.NewReader("")},
		{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
		{Key: "originaldir/empty/", Body: strings.NewReader("")},
		{Key: "originaldir/subdir/", Body: strings.NewReader("")},
		{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
		{Key: "originaldir/subdir/originaldir/more-nested.txt", Body: strings.NewReader("more nested content")},
	})

	rootDir := testutil.FakeRootDirectory(t)

	originalDir := testutil.NewDirectory(t, "originaldir", rootDir.Path())
	testutil.AddFileToDirectory(t, originalDir, "file.txt")
	testutil.AddSubDirectoryToDirectory(t, originalDir, "empty")
	subDir := testutil.AddSubDirectoryToDirectory(t, originalDir, "subdir")
	testutil.AddFileToDirectory(t, subDir, "nested.txt")
	subSubDir := testutil.AddSubDirectoryToDirectory(t, subDir, "originaldir")
	testutil.AddFileToDirectory(t, subSubDir, "more-nested.txt")

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should ask for user validation before renaming a non-empty directory", func(t *testing.T) {
		// Given
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

		inputEvt := directory.NewRenamedEvent(originalDir, "newname")

		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.UserValidationEvent)
				assert.True(t, ok)
				assert.Equal(t, inputEvt, e.Reason())
				close(done)
			}).
			Times(1)

		_, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "newname/file.txt")
	})

	t.Run("should rename directory and its content after user had validated it", func(t *testing.T) {
		// Given
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
			Times(2)

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

		_, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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

		oldKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "originaldir/")
		assert.Len(t, oldKeys, 0)

		resKeys := listKeys(t, client, testutil.FakeS3LikeBucketName, "newname/")
		assert.Len(t, resKeys, 5)

		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/file.txt", "file content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/empty/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/", "")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/nested.txt", "nested content")
		assertObjectContent(t, client, testutil.FakeS3LikeBucketName, "newname/subdir/originaldir/more-nested.txt", "more nested content")

		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir/file.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir/empty/")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir/subdir/")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir/subdir/nested.txt")
		assertObjectNotExists(t, client, testutil.FakeS3LikeBucketName, "originaldir/subdir/originaldir/more-nested.txt")
	})

	//t.Run("should rename empty directory directly without validation", func(t *testing.T) {
	//	// Given
	//	ctrl := gomock.NewController(t)
	//	mockBus := mocks_event.NewMockBus(ctrl)
	//	mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
	//	mockNotifRepo := mocks_notification.NewMockRepository(ctrl)
	//
	//	mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)
	//
	//	// Create an empty directory
	//	emptyDir, err := directory.New(fakeConnID, "emptydir", directory.RootPath)
	//	require.NoError(t, err)
	//
	//	fakeEventChan := make(chan event.Event, 1)
	//	defer close(fakeEventChan)
	//
	//	mockBus.EXPECT().
	//		Subscribe().
	//		Return(event.NewSubscriber(fakeEventChan))
	//
	//	mockConnRepo.EXPECT().
	//		Get(gomock.AssignableToTypeOf(testutil.CtxType)).
	//		Return(fakeDeck, nil).
	//		Times(2) // Once for directory creation, once for rename
	//
	//	done := make(chan struct{})
	//	// Expect directory creation success
	//	mockBus.EXPECT().
	//		Publish(gomock.Any()).
	//		Do(func(evt event.Event) {
	//			_, ok := evt.(directory.CreatedSuccessEvent)
	//			assert.True(t, ok)
	//		}).
	//		Times(1)
	//
	//	// Expect directory rename success (no validation event)
	//	mockBus.EXPECT().
	//		Publish(gomock.Cond(func(evt event.Event) bool {
	//			e, ok := evt.(directory.RenamedSuccessEvent)
	//			res := assert.True(t, ok) &&
	//				assert.Equal(t, "renamed_emptydir", e.Directory().Name())
	//			close(done)
	//			return res
	//		})).
	//		Times(1)
	//
	//	_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
	//	require.NoError(t, err)
	//
	//	// First, create the empty directory
	//	fakeEventChan <- directory.NewCreatedEvent(rootDir, emptyDir)
	//	// Wait a bit for the directory to be created
	//	time.Sleep(500 * time.Millisecond)
	//
	//	// When
	//	oldPath := emptyDir.Path()
	//	fakeEventChan <- directory.NewRenamedEvent(emptyDir, oldPath, "renamed_emptydir")
	//	assert.Eventually(t, func() bool {
	//		select {
	//		case <-done:
	//			return true
	//		default:
	//			return false
	//		}
	//	}, 5*time.Second, 100*time.Millisecond)
	//
	//	// Verify new directory exists
	//	_, err = client.GetObject(ctx, &s3.GetObjectInput{
	//		Bucket: aws.String(bucketName),
	//		Key:    aws.String("renamed_emptydir/"),
	//	})
	//	require.NoError(t, err)
	//})
	//
	//t.Run("should handle rename failure gracefully", func(t *testing.T) {
	//	// Given
	//	ctrl := gomock.NewController(t)
	//	mockBus := mocks_event.NewMockBus(ctrl)
	//	mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
	//	mockNotifRepo := mocks_notification.NewMockRepository(ctrl)
	//
	//	mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(1)
	//
	//	originalDir, err := directory.New(fakeConnID, "originaldir", directory.RootPath)
	//	require.NoError(t, err)
	//
	//	fakeEventChan := make(chan event.Event, 1)
	//	defer close(fakeEventChan)
	//
	//	mockBus.EXPECT().
	//		Subscribe().
	//		Return(event.NewSubscriber(fakeEventChan))
	//
	//	mockConnRepo.EXPECT().
	//		Get(gomock.AssignableToTypeOf(testutil.CtxType)).
	//		Return(fakeDeck, nil).
	//		Times(1)
	//
	//	done := make(chan struct{})
	//	// Expect rename failure event
	//	mockBus.EXPECT().
	//		Publish(gomock.Cond(func(evt event.Event) bool {
	//			_, ok := evt.(directory.RenamedFailureEvent)
	//			close(done)
	//			return ok
	//		})).
	//		Times(1)
	//
	//	_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
	//	require.NoError(t, err)
	//
	//	// When - try to rename to an existing directory name
	//	oldPath := originalDir.Path()
	//	fakeEventChan <- directory.NewRenamedEvent(originalDir, oldPath, "mydir") // mydir already exists
	//	assert.Eventually(t, func() bool {
	//		select {
	//		case <-done:
	//			return true
	//		default:
	//			return false
	//		}
	//	}, 5*time.Second, 100*time.Millisecond)
	//})
}
