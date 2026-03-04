package infrastructure_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	mocks_connection_deck "github.com/thomas-marquis/s3-box/mocks/connection_deck"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

const (
	fakeAccessKeyId     = "AZERTY"
	fakeSecretAccessKey = "dfhdh2432J4bbhjkb"
)

var (
	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func setupS3testContainer(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()

	lsContainer, err := localstack.Run(ctx, "localstack/localstack:3.0")
	require.NoError(t, err)

	endpoint, err := lsContainer.PortEndpoint(ctx, "4566", "")
	require.NoError(t, err)

	return endpoint, func() {
		_ = lsContainer.Terminate(ctx)
	}
}

func setupS3Client(t *testing.T, endpoint string) *s3.Client {
	t.Helper()

	awsCfg := aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://" + endpoint),
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return s3Client
}

type fakeS3Object struct {
	Key  string
	Body io.Reader
}

func setupS3Bucket(ctx context.Context, t *testing.T, client *s3.Client, bucketName string, content []fakeS3Object) {
	t.Helper()

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err)

	for _, obj := range content {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(obj.Key),
			Body:   obj.Body,
		})
		require.NoError(t, err)
	}
}

func getObject(t *testing.T, client *s3.Client, bucketName, key string) io.ReadCloser {
	t.Helper()

	res, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	require.NoError(t, err)

	return res.Body
}

func putObject(t *testing.T, client *s3.Client, bucketName, key string, body io.Reader) {
	t.Helper()

	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   body,
	})
	require.NoError(t, err)
}

func TestS3DirectoryRepository_loadDirectory(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	bucketName := "test-bucket"

	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{
		{Key: "root_file.txt"},
		{Key: "mydir/"},
		{Key: "mydir/file_in_dir.txt"},
	})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))

	t.Run("should publish root directory and its content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		rootDir, err := directory.New(fakeConnID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

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
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(1)

		_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
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
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(1)

		dir, err := directory.New(fakeConnID, "mydir", directory.RootPath)
		require.NoError(t, err)

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

		awsConnID := connection_deck.NewConnectionID()
		awsDeck := connection_deck.New()
		awsDeck.New("AWS connection", fakeAccessKeyId, fakeSecretAccessKey, "any-bucket",
			connection_deck.AsAWS("us-east-1"),
			connection_deck.WithID(awsConnID))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(ctxType)).
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
						"api error PermanentRedirect: The bucket you are attempting to access must be addressed using the specified endpoint.")
				close(done)
				return res
			})).
			Times(1)

		dir, err := directory.New(awsConnID, directory.RootDirName, directory.NilParentPath)
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

	bucketName := "test-bucket"

	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{
		{Key: "root_file.txt", Body: strings.NewReader("coucou")},
		{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("lolo")},
	})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))

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
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("root_file.txt", directory.RootPath)
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), fakeConnID, file)

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
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("file_in_dir.txt", directory.NewPath("/mydir/"))
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), fakeConnID, file)

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

	bucketName := "test-bucket"

	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{
		{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("download-me")},
	})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))

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
			Get(gomock.AssignableToTypeOf(ctxType)).
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
		fakeEventChan <- directory.NewContentDownloadedEvent(fakeConnID, content)
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
			Get(gomock.AssignableToTypeOf(ctxType)).
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

		// When
		fakeEventChan <- directory.NewContentDownloadedEvent(fakeConnID, content)
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

	bucketName := "test-bucket"
	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))

	t.Run("should create an empty file", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		dir, err := directory.New(fakeConnID, "mydir", directory.RootPath)
		require.NoError(t, err)
		newFile, err := directory.NewFile("new_file.txt", dir.Path())
		require.NoError(t, err)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(ctxType)).
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
		fakeEventChan <- directory.NewFileCreatedEvent(fakeConnID, dir, newFile)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		remoteObj := getObject(t, client, bucketName, "mydir/new_file.txt")
		defer remoteObj.Close() // nolint:errcheck
		downloaded, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "", string(downloaded))
	})
}

func TestNewS3DirectoryRepository_renameFile(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	bucketName := "test-bucket"
	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{
		{Key: "mydir/original.txt", Body: strings.NewReader("original content")},
	})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))

	t.Run("should rename a file successfully", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		// Don't expect any error notifications for the test
		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		parentDir, err := directory.New(fakeConnID, "mydir", directory.RootPath)
		require.NoError(t, err)

		renamedFile, err := directory.NewFile("renamed.txt", parentDir.Path())
		require.NoError(t, err)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(2) // Once for file creation, once for rename

		done := make(chan struct{})
		// Expect file creation success
		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				_, ok := evt.(directory.FileCreatedSuccessEvent)
				return assert.True(t, ok)
			})).
			Times(1)

		// Expect file rename success
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

		// First, create the original file to ensure it exists
		originalFile, err := directory.NewFile("original.txt", parentDir.Path())
		require.NoError(t, err)
		fakeEventChan <- directory.NewFileCreatedEvent(fakeConnID, parentDir, originalFile)
		// Wait a bit for the file to be created
		time.Sleep(500 * time.Millisecond)

		// When
		oldName := directory.FileName("original.txt")
		fakeEventChan <- directory.NewFileRenamedEvent(parentDir, renamedFile, oldName)
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		// Verify old file is gone
		_, err = client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String("mydir/original.txt"),
		})
		assert.Error(t, err)

		// Verify new file exists with correct content
		remoteObj := getObject(t, client, bucketName, "mydir/renamed.txt")
		defer remoteObj.Close() // nolint:errcheck
		downloaded, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "original content", string(downloaded))
	})
}

// TestNewS3DirectoryRepository_renameDirectory tests the complete directory rename functionality
// including S3 object copying, deletion, and directory marker creation.
// This test was added as part of Phase 6 integration testing to ensure proper event propagation
// and infrastructure layer functionality for directory rename operations.
func TestNewS3DirectoryRepository_renameDirectory(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	bucketName := "test-bucket"
	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{
		{Key: "originaldir/", Body: strings.NewReader("")},
		{Key: "originaldir/file.txt", Body: strings.NewReader("file content")},
		{Key: "originaldir/subdir/", Body: strings.NewReader("")},
		{Key: "originaldir/subdir/nested.txt", Body: strings.NewReader("nested content")},
	})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))

	t.Run("should rename a directory successfully", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		// Don't expect any error notifications for the test
		mockNotifRepo.EXPECT().NotifyError(gomock.Any()).Times(0).MaxTimes(0)

		rootDir, err := directory.New(fakeConnID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		originalDir, err := directory.New(fakeConnID, "originaldir", directory.RootPath)
		require.NoError(t, err)

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)

		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(2) // Once for directory creation, once for rename

		done := make(chan struct{})
		// Expect directory creation success
		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				_, ok := evt.(directory.CreatedSuccessEvent)
				assert.True(t, ok)
			}).
			Times(1)

		// Expect directory rename success
		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				e, ok := evt.(directory.RenamedSuccessEvent)
				assert.True(t, ok)
				assert.Equal(t, "renameddir", e.Directory().Name())
				close(done)
			}).
			Times(1)

		// Expect load event for the renamed directory (triggered by the infrastructure layer)
		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				_, ok := evt.(directory.LoadEvent)
				assert.True(t, ok)
			}).
			Times(1)

		_, err = infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// First, create the original directory to ensure it exists
		fakeEventChan <- directory.NewCreatedEvent(rootDir, originalDir)
		// Wait a bit for the directory to be created
		time.Sleep(500 * time.Millisecond)

		// When
		oldPath := originalDir.Path()
		fakeEventChan <- directory.NewRenamedEvent(originalDir, oldPath, "renameddir")
		assert.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, 5*time.Second, 100*time.Millisecond)

		// Verify new directory exists (this implies the old one is gone)
		_, err = client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String("renameddir/"),
		})
		require.NoError(t, err)

		// Verify file was moved to new directory
		remoteObj := getObject(t, client, bucketName, "renameddir/file.txt")
		defer remoteObj.Close() // nolint:errcheck
		downloaded, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "file content", string(downloaded))

		// Verify nested directory was also moved
		_, err = client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String("renameddir/subdir/"),
		})
		require.NoError(t, err)

		// Verify nested file was moved with correct content
		remoteObj2 := getObject(t, client, bucketName, "renameddir/subdir/nested.txt")
		defer remoteObj2.Close() // nolint:errcheck
		downloaded2, err := io.ReadAll(remoteObj2)
		require.NoError(t, err)
		assert.Equal(t, "nested content", string(downloaded2))
	})
}
