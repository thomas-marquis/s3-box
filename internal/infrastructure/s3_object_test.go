package infrastructure_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
)

func TestS3Object_Read(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	downloader := manager.NewDownloader(client)
	uploader := manager.NewUploader(client)

	bucketName := "test-bucket-read"
	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{
		{Key: "existing-file.txt", Body: strings.NewReader("hello world")},
	})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))
	conn, err := fakeDeck.GetByID(fakeConnID)
	require.NoError(t, err)

	t.Run("should read the object content when exists", func(t *testing.T) {
		// Given
		file, err := directory.NewFile("existing-file.txt", directory.RootPath)
		require.NoError(t, err)

		obj, err := infrastructure.NewS3Object(ctx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		n, err := obj.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		content, err := io.ReadAll(obj)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "hello world", string(content))
	})

	t.Run("should read the object content when exists with non-zero offset", func(t *testing.T) {
		// Given
		file, err := directory.NewFile("existing-file.txt", directory.RootPath)
		require.NoError(t, err)

		obj, err := infrastructure.NewS3Object(ctx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		n, err := obj.Seek(6, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(6), n)

		content, err := io.ReadAll(obj)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "world", string(content))
	})

	t.Run("should return an error when the object does not exists", func(t *testing.T) {
		// Given
		file, err := directory.NewFile("non-existing-file.txt", directory.RootPath)
		require.NoError(t, err)

		obj, err := infrastructure.NewS3Object(ctx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		buf := make([]byte, 100)
		n, err := obj.Read(buf)

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object does not exist")
		assert.Equal(t, 0, n)
	})
}

func TestS3Object_Write(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	downloader := manager.NewDownloader(client)
	uploader := manager.NewUploader(client)

	bucketName := "test-bucket-write"
	setupS3Bucket(ctx, t, client, bucketName, []fakeS3Object{})

	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, bucketName,
		connection_deck.AsS3Like(endpoint, false),
		connection_deck.WithID(fakeConnID))
	conn, err := fakeDeck.GetByID(fakeConnID)
	require.NoError(t, err)

	t.Run("should create the object if not exists then makes it readable", func(t *testing.T) {
		// Given
		fileKey := "brand-new-file.txt"
		file, err := directory.NewFile(fileKey, directory.RootPath)
		require.NoError(t, err)

		obj, err := infrastructure.NewS3Object(ctx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		n, err := obj.Write([]byte("new content"))

		// Then
		require.NoError(t, err)
		assert.Equal(t, 11, n)

		obj.Seek(0, io.SeekStart)
		localContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(localContent))

		remoteObj := getObject(t, client, bucketName, fileKey)
		defer remoteObj.Close()

		uploadedContent, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(uploadedContent))
	})

	t.Run("should append to the object's content if exists", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-0.txt"
		putObject(t, client, bucketName, fileKey, strings.NewReader("initial content"))

		file, err := directory.NewFile(fileKey, directory.RootPath)
		require.NoError(t, err)

		obj, err := infrastructure.NewS3Object(ctx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		n, err := obj.Write([]byte(" appended"))

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 9, n)

		obj.Seek(0, io.SeekStart)
		localContent, err := io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content appended", string(localContent))

		// Verify the content was updated in S3
		remoteObj := getObject(t, client, bucketName, fileKey)
		defer remoteObj.Close()

		uploadedContent, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "initial content appended", string(uploadedContent))
	})

	t.Run("should overwrite the object's content if exists and after seeking to 0", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-1.txt"
		putObject(t, client, bucketName, fileKey, strings.NewReader("initial content"))

		file, err := directory.NewFile(fileKey, directory.RootPath)
		require.NoError(t, err)

		obj, err := infrastructure.NewS3Object(ctx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		n, err := obj.Seek(0, io.SeekStart)
		n2, err2 := fmt.Fprint(obj, "New content")
		obj.Seek(0, io.SeekStart)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.NoError(t, err2)
		assert.Equal(t, 11, n2)

		remoteObj := getObject(t, client, bucketName, fileKey)
		defer remoteObj.Close()
		remoteContent, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "New content", string(remoteContent))

		localContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, "New content", string(localContent))
	})

	t.Run("should reset the object content and offset on error when the file exists", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-2.txt"
		putObject(t, client, bucketName, fileKey, strings.NewReader("initial content"))

		file, err := directory.NewFile(fileKey, directory.RootPath)
		require.NoError(t, err)

		newCtx, cancel := context.WithCancel(ctx)
		obj, err := infrastructure.NewS3Object(newCtx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		_, err = obj.Seek(0, io.SeekStart)
		require.NoError(t, err)

		// simulate a server error, then write
		cancel()
		_, err = obj.Write([]byte("should not be written"))

		// Then
		assert.Error(t, err)

		localContent, err := io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content", string(localContent))

		remoteObj := getObject(t, client, bucketName, fileKey)
		defer remoteObj.Close()
		remoteContent, err := io.ReadAll(remoteObj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content", string(remoteContent))
	})

	t.Run("should reset the object content and offset on error with a non-zero offset", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-3.txt"

		putObject(t, client, bucketName, fileKey, strings.NewReader("initial content"))

		file, err := directory.NewFile(fileKey, directory.RootPath)
		require.NoError(t, err)

		newCtx, cancel := context.WithCancel(ctx)
		obj, err := infrastructure.NewS3Object(newCtx, downloader, uploader, conn, file)
		require.NoError(t, err)

		// When
		_, err = obj.Seek(int64(len("initial ")), io.SeekStart)
		require.NoError(t, err)

		// simulate a server error, then write
		cancel()
		_, err = obj.Write([]byte("new"))

		// Then
		assert.Error(t, err)

		localContent, err := io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "content", string(localContent))

		obj.Seek(0, io.SeekStart)
		localContent, err = io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content", string(localContent))

		remoteObj := getObject(t, client, bucketName, fileKey)
		defer remoteObj.Close()
		remoteContent, err := io.ReadAll(remoteObj)
		require.NoError(t, err)
		assert.Equal(t, "initial content", string(remoteContent))
	})
}
