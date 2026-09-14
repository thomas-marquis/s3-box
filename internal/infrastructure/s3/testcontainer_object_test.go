package s3_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
	"github.com/thomas-marquis/s3-box/internal/tu"
)

func TestS3ObjectLazyReadOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := tu.SetupS3testContainer(ctx, t)
	defer terminate()
	testClient := tu.SetupS3Client(t, endpoint)

	bucket := tu.FakeS3LikeBucketName
	largeContent := strings.Repeat("ABCDEFGHIJ", 1000) // 10KB of data
	tu.SetupS3Bucket(ctx, t, testClient, bucket, []tu.FakeS3Object{
		{Key: "large-file.csv", Body: strings.NewReader(largeContent)},
	})
	conn := tu.FakeAwsConnectionWithEndpoint(t, endpoint, bucket)
	client := s3client.NewAwsClient(conn)

	t.Run("should read first chunk without loading full file", func(t *testing.T) {
		// Given
		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile("large-file.csv", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewLazyObject(ctx, client, file)
		require.NoError(t, err)

		// When: read just the first 100 bytes
		buf := make([]byte, 100)
		n, err := obj.Read(buf)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 100, n)
		assert.Equal(t, largeContent[:100], string(buf[:n]))
	})

	t.Run("should seek and read different parts of file", func(t *testing.T) {
		// Given
		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile("large-file.csv", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewLazyObject(ctx, client, file)
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
		assert.Equal(t, largeContent[500:600], string(buf[:n]))
	})

	t.Run("should handle read-only mode and return error on write", func(t *testing.T) {
		// Given
		rootDir, err := directory.NewRoot(conn.ID())
		require.NoError(t, err)
		file, err := directory.NewFile("large-file.csv", rootDir)
		require.NoError(t, err)

		obj, err := s3.NewLazyObject(ctx, client, file)
		require.NoError(t, err)

		// When
		n, err := obj.Write([]byte("test"))

		// Then
		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "write not supported in lazy read-only mode", err.Error())
	})
}
