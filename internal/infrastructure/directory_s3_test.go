package infrastructure_test

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

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

func TestS3DirectoryRepository_GetByPath(t *testing.T) {
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

	t.Run("should returns root directory and its content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(fakeEventChan)

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		dir, err := repo.GetByPath(ctx, fakeConnID, directory.RootPath)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.RootDirName, dir.Name())
		assert.Equal(t, directory.RootPath, dir.Path())

		// Expect 1 file and 1 subdirectory
		assert.Len(t, dir.Files(), 1)
		assert.Equal(t, "root_file.txt", dir.Files()[0].Name().String())

		assert.Len(t, dir.SubDirectories(), 1)
		assert.Equal(t, "/mydir/", dir.SubDirectories()[0].String())
	})

	t.Run("should returns subdirectory and its content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(fakeEventChan)

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(ctxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		subPath := directory.NewPath("/mydir/")

		// When
		dir, err := repo.GetByPath(ctx, fakeConnID, subPath)

		// Then
		require.NoError(t, err)

		assert.Equal(t, "mydir", dir.Name())
		assert.Equal(t, subPath, dir.Path())

		// Expect 1 file
		assert.Len(t, dir.Files(), 1)
		assert.Equal(t, "file_in_dir.txt", dir.Files()[0].Name().String())
		assert.Len(t, dir.SubDirectories(), 0)
	})

	t.Run("should handle AWS connection without custom endpoint", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(fakeEventChan).
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

		repo, err := infrastructure.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		// When
		_, err = repo.GetByPath(ctx, awsConnID, directory.RootPath)

		// Then
		// It should fail with a connection error since we don't have real AWS access,
		// but we want to ensure the baseEp was nil, which leads to using default AWS endpoints.
		assert.Error(t, err)
		assert.Contains(t, err.Error(),
			"api error PermanentRedirect: The bucket you are attempting to access must be addressed using the specified endpoint.")
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
			Return(fakeEventChan)

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
			Return(fakeEventChan)

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
