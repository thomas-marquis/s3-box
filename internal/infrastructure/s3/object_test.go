package s3_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestS3Object_Read(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	testClient := testutil.SetupS3Client(t, endpoint)

	bucket := testutil.FakeS3LikeBucketName
	testutil.SetupS3Bucket(ctx, t, testClient, bucket, []testutil.FakeS3Object{
		{Key: "existing-file.txt", Body: strings.NewReader("hello world")},
	})
	conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucket)
	client := s3client.NewAwsClient(conn)

	t.Run("should read the object content when exists", func(t *testing.T) {
		// Given
		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile("existing-file.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
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
		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile("existing-file.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
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
		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile("non-existing-file.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
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
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	testClient := testutil.SetupS3Client(t, endpoint)

	bucket := testutil.FakeS3LikeBucketName
	testutil.SetupS3Bucket(ctx, t, testClient, bucket, []testutil.FakeS3Object{})
	conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucket)
	client := s3client.NewAwsClient(conn)

	t.Run("should create the object if not exists then makes it readable", func(t *testing.T) {
		// Given
		fileKey := "brand-new-file.txt"
		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
		require.NoError(t, err)

		// When
		n, err := obj.Write([]byte("new content"))

		// Then
		require.NoError(t, err)
		assert.Equal(t, 11, n)

		obj.Seek(0, io.SeekStart) // nolint:errcheck
		localContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(localContent))

		testutil.AssertObjectContent(t, testClient, testutil.FakeS3LikeBucketName, fileKey, "new content")
	})

	t.Run("should append to the object's content if exists", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-0.txt"
		testutil.PutObject(t, testClient, testutil.FakeS3LikeBucketName, fileKey, strings.NewReader("initial content"))

		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
		require.NoError(t, err)

		// When
		n, err := obj.Write([]byte(" appended"))

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 9, n)

		obj.Seek(0, io.SeekStart) // nolint:errcheck
		localContent, err := io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content appended", string(localContent))

		// Verify the content was updated in S3
		testutil.AssertObjectContent(t, testClient, testutil.FakeS3LikeBucketName, fileKey, "initial content appended")
	})

	t.Run("should overwrite the object's content if exists and after seeking to 0", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-1.txt"
		testutil.PutObject(t, testClient, testutil.FakeS3LikeBucketName, fileKey, strings.NewReader("initial content"))

		rootDir, err := directory.NewRoot(testutil.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
		require.NoError(t, err)

		// When
		n, err := obj.Seek(0, io.SeekStart)
		n2, err2 := fmt.Fprint(obj, "New content")
		obj.Seek(0, io.SeekStart) // nolint:errcheck

		// Then
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.NoError(t, err2)
		assert.Equal(t, 11, n2)

		testutil.AssertObjectContent(t, testClient, testutil.FakeS3LikeBucketName, fileKey, "New content")

		localContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, "New content", string(localContent))
	})

	t.Run("should reset the object content and offset on error when the file exists", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists.txt"

		bucketName := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(context.TODO(), t, testClient, bucketName, []testutil.FakeS3Object{
			{Key: fileKey, Body: strings.NewReader("initial content")},
		})
		conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
		client := s3client.NewAwsClient(conn, func(options *awsS3.Options) {
			options.Interceptors.AddBeforeTransmit(&fakeErrorInterceptor{
				PutErrorForKeys: []string{fileKey},
			})
		})

		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(context.TODO(), client, file)
		require.NoError(t, err)

		// When
		_, err = obj.Seek(0, io.SeekStart)
		require.NoError(t, err)

		_, err = obj.Write([]byte("should not be written"))

		// Then
		assert.Error(t, err)

		localContent, err := io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content", string(localContent))

		testutil.AssertObjectContent(t, testClient, bucketName, fileKey, "initial content")
	})

	t.Run("should reset the object content and offset on error with a non-zero offset", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists.txt"

		bucketName := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(context.TODO(), t, testClient, bucketName, []testutil.FakeS3Object{
			{Key: fileKey, Body: strings.NewReader("initial content")},
		})
		conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
		client := s3client.NewAwsClient(conn, func(options *awsS3.Options) {
			options.Interceptors.AddBeforeTransmit(&fakeErrorInterceptor{
				PutErrorForKeys: []string{fileKey},
			})
		})

		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file)
		require.NoError(t, err)

		// When
		_, err = obj.Seek(int64(len("initial ")), io.SeekStart)
		require.NoError(t, err)

		// simulate a server error, then write
		_, err = obj.Write([]byte("new"))

		// Then
		assert.Error(t, err)

		localContent, err := io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "content", string(localContent))

		obj.Seek(0, io.SeekStart) // nolint:errcheck
		localContent, err = io.ReadAll(obj)
		assert.NoError(t, err)
		assert.Equal(t, "initial content", string(localContent))

		testutil.AssertObjectContent(t, testClient, bucketName, fileKey, "initial content")
	})
}
