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
	"github.com/thomas-marquis/s3-box/internal/tu"
	mocks_s3client "github.com/thomas-marquis/s3-box/mocks/infrastructure"
	"go.uber.org/mock/gomock"
)

func TestS3Object_Read(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	largeContent := []byte(strings.Repeat("ABCDEFGHIJ", 1000)) // 10KB of data
	contentLen := len(largeContent)

	endpoint, terminate := tu.SetupS3testContainer(ctx, t)
	defer terminate()
	testClient := tu.SetupS3Client(t, endpoint)

	bucket := tu.FakeS3LikeBucketName
	tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
		{Key: "existing-file.txt", Body: strings.NewReader("hello world")},
		{Key: "large-file.csv", Body: strings.NewReader(string(largeContent))},
	})
	conn := tu.FakeAwsConnectionWithEndpoint(t, endpoint, bucket)
	client := s3client.NewAwsClient(conn)

	rootDir, err := directory.NewRoot(conn.ID())
	require.NoError(t, err)
	largeFile, err := directory.NewFile("large-file.csv", rootDir, directory.WithFileSize(uint64(contentLen)))
	require.NoError(t, err)

	t.Run("should read the object content when exists", func(t *testing.T) {
		// Given
		rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile("existing-file.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
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
		rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile("existing-file.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
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
		rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile("non-existing-file.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
		require.NoError(t, err)

		// When
		buf := make([]byte, 100)
		n, err := obj.Read(buf)

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object does not exist")
		assert.Equal(t, 0, n)
	})

	t.Run("should read first chunk without loading full file when lazy loaded", func(t *testing.T) {
		// Given
		obj, err := s3.NewObject(ctx, client, largeFile, true)
		require.NoError(t, err)

		// When: read just the first 100 bytes
		buf := make([]byte, 100)
		n, err := obj.Read(buf)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 100, n)
		assert.Equal(t, string(largeContent[:100]), string(buf[:n]))
	})

	t.Run("should seek and read different parts of file when lazy loaded", func(t *testing.T) {
		// Given
		obj, err := s3.NewObject(ctx, client, largeFile, true)
		require.NoError(t, err)

		// When: seek to position 500 and read 100 bytes
		pos, err := obj.Seek(500, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(500), pos)

		buf := make([]byte, 100)
		n, err := obj.Read(buf)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 100, n)
		assert.Equal(t, string(largeContent[500:600]), string(buf[:n]))
	})

	t.Run("should fetch continuous ranges with different lengths when lazy loaded", func(t *testing.T) {
		// Given
		obj, err := s3.NewObject(ctx, client, largeFile, true)
		require.NoError(t, err)

		// When: first read 100 bytes
		buf1 := make([]byte, 100)
		n1, err := obj.Read(buf1)
		require.NoError(t, err)
		assert.Equal(t, 100, n1)
		assert.Equal(t, string(largeContent[:100]), string(buf1[:n1]))

		// When: second read 50 bytes (different length)
		buf2 := make([]byte, 50)
		n2, err := obj.Read(buf2)
		require.NoError(t, err)
		assert.Equal(t, 50, n2)
		assert.Equal(t, string(largeContent[100:150]), string(buf2[:n2]))

		// When: third read 200 bytes (different length again)
		buf3 := make([]byte, 200)
		n3, err := obj.Read(buf3)
		require.NoError(t, err)
		assert.Equal(t, 200, n3)
		assert.Equal(t, string(largeContent[150:350]), string(buf3[:n3]))
	})

	t.Run("should cache data and not re-fetch on rewind when lazy loaded", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockClient := mocks_s3client.NewMockClient(ctrl)

		gomock.InOrder(
			mockClient.EXPECT().
				GetObject(gomock.AssignableToTypeOf(tu.CtxType), gomock.Eq("large-file.csv"), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, opts ...s3client.Option) (*awsS3.GetObjectOutput, error) {
					mockIn := &awsS3.GetObjectInput{}
					for _, o := range opts {
						o(mockIn)
					}
					assert.Equal(t, "bytes=0-99", *mockIn.Range)
					return &awsS3.GetObjectOutput{
						Body: io.NopCloser(strings.NewReader(string(largeContent[:100]))),
					}, nil
				}).
				Times(1),
			mockClient.EXPECT().
				GetObject(gomock.AssignableToTypeOf(tu.CtxType), gomock.Eq("large-file.csv"), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, opts ...s3client.Option) (*awsS3.GetObjectOutput, error) {
					mockIn := &awsS3.GetObjectInput{}
					for _, o := range opts {
						o(mockIn)
					}
					assert.Equal(t, "bytes=100-199", *mockIn.Range)
					return &awsS3.GetObjectOutput{
						Body: io.NopCloser(strings.NewReader(string(largeContent[100:200]))),
					}, nil
				}).
				Times(1),
		)

		obj, err := s3.NewObject(ctx, mockClient, largeFile, true)
		require.NoError(t, err)

		// When: read first 100 bytes
		buf1 := make([]byte, 100)
		n1, err := obj.Read(buf1)
		require.NoError(t, err)
		assert.Equal(t, 100, n1)
		firstChunk := string(buf1[:n1])

		// When: read next 100 bytes
		buf2 := make([]byte, 100)
		n2, err := obj.Read(buf2)
		require.NoError(t, err)
		assert.Equal(t, 100, n2)
		secondChunk := string(buf2[:n2])

		// When: seek back to start
		pos, err := obj.Seek(0, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(0), pos)

		// When: read first 100 bytes again (should come from cache, not S3)
		buf3 := make([]byte, 100)
		n3, err := obj.Read(buf3)
		require.NoError(t, err)
		assert.Equal(t, 100, n3)
		assert.Equal(t, firstChunk, string(buf3[:n3]))

		// When: read next 100 bytes again (should also come from cache)
		buf4 := make([]byte, 100)
		n4, err := obj.Read(buf4)
		require.NoError(t, err)
		assert.Equal(t, 100, n4)
		assert.Equal(t, secondChunk, string(buf4[:n4]))
	})
}

func TestS3Object_Write(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := tu.SetupS3testContainer(ctx, t)
	defer terminate()
	testClient := tu.SetupS3Client(t, endpoint)

	bucket := tu.FakeS3LikeBucketName
	tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{})
	conn := tu.FakeAwsConnectionWithEndpoint(t, endpoint, bucket)
	client := s3client.NewAwsClient(conn)

	t.Run("should create the object if not exists then makes it readable", func(t *testing.T) {
		// Given
		fileKey := "brand-new-file.txt"
		rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
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

		tu.AssertObjectContent(t, testClient, tu.FakeS3LikeBucketName, fileKey, "new content")
	})

	t.Run("should append to the object's content if exists", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-0.txt"
		tu.PutObject(t, testClient, tu.FakeS3LikeBucketName, fileKey, strings.NewReader("initial content"))

		rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
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
		tu.AssertObjectContent(t, testClient, tu.FakeS3LikeBucketName, fileKey, "initial content appended")
	})

	t.Run("should overwrite the object's content if exists and after seeking to 0", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists-1.txt"
		tu.PutObject(t, testClient, tu.FakeS3LikeBucketName, fileKey, strings.NewReader("initial content"))

		rootDir, err := directory.NewRoot(tu.FakeAwsConnectionId)
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
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

		tu.AssertObjectContent(t, testClient, tu.FakeS3LikeBucketName, fileKey, "New content")

		localContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, "New content", string(localContent))
	})

	t.Run("should reset the object content and offset on error when the file exists", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists.txt"

		bucketName := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(context.TODO(), t, testClient, bucketName, []tu.FakeS3Object{
			{Key: fileKey, Body: strings.NewReader("initial content")},
		})
		conn := tu.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
		client := s3client.NewAwsClient(conn, func(options *awsS3.Options) {
			options.Interceptors.AddBeforeTransmit(&fakeErrorInterceptor{
				PutErrorForKeys: []string{fileKey},
			})
		})

		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(context.TODO(), client, file, false)
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

		tu.AssertObjectContent(t, testClient, bucketName, fileKey, "initial content")
	})

	t.Run("should reset the object content and offset on error with a non-zero offset", func(t *testing.T) {
		// Given
		fileKey := "this-file-exists.txt"

		bucketName := tu.FakeRandomBucketName()
		tu.SetupS3Bucket(context.TODO(), t, testClient, bucketName, []tu.FakeS3Object{
			{Key: fileKey, Body: strings.NewReader("initial content")},
		})
		conn := tu.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
		client := s3client.NewAwsClient(conn, func(options *awsS3.Options) {
			options.Interceptors.AddBeforeTransmit(&fakeErrorInterceptor{
				PutErrorForKeys: []string{fileKey},
			})
		})

		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile(fileKey, rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, false)
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

		tu.AssertObjectContent(t, testClient, bucketName, fileKey, "initial content")
	})

	t.Run("should handle read-only mode and return error on write when lazy loaded", func(t *testing.T) {
		// Given
		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile("this-file-exists.txt", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewObject(ctx, client, file, true)
		require.NoError(t, err)

		// When
		n, err := obj.Write([]byte("test"))

		// Then
		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "write not supported in lazy read-only mode", err.Error())
	})
}
