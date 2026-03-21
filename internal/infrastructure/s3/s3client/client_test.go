package s3client_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestClient_RenameObject(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()

	s3Client := testutil.SetupS3Client(t, endpoint)
	bucketName := testutil.FakeRandomBucketName()
	oldKey := "old-file.txt"
	newKey := "new-file.txt"
	content := "hello world"

	testutil.SetupS3Bucket(ctx, t, s3Client, bucketName, []testutil.FakeS3Object{
		{Key: oldKey, Body: strings.NewReader(content)},
	})

	conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
	client := s3client.NewAwsClient(conn, func(o *s3.Options) {
		o.Region = "us-east-1"
	})

	t.Run("should rename object successfully", func(t *testing.T) {
		// When
		err := client.RenameObject(ctx, oldKey, newKey)

		// Then
		assert.NoError(t, err)
		testutil.AssertObjectContent(t, s3Client, bucketName, newKey, content)
		testutil.AssertObjectNotExists(t, s3Client, bucketName, oldKey)
	})

	t.Run("should return error if old key does not exist", func(t *testing.T) {
		// When
		err := client.RenameObject(ctx, "non-existent.txt", "target.txt")

		// Then
		assert.Error(t, err)
	})
}

func TestClient_ListObjects(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()

	s3Client := testutil.SetupS3Client(t, endpoint)
	bucketName := testutil.FakeRandomBucketName()

	testutil.SetupS3Bucket(ctx, t, s3Client, bucketName, []testutil.FakeS3Object{
		{Key: "dir1/file1.txt", Body: strings.NewReader("1")},
		{Key: "dir1/file2.txt", Body: strings.NewReader("22")},
		{Key: "dir2/file3.txt", Body: strings.NewReader("333")},
		{Key: "file4.txt", Body: strings.NewReader("4444")},
	})

	conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
	client := s3client.NewAwsClient(conn, func(o *s3.Options) {
		o.Region = "us-east-1"
	})

	t.Run("should list objects non-recursively in root", func(t *testing.T) {
		// When
		res, err := client.ListObjects(ctx, "", false)

		// Then
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"file4.txt"}, res.Keys)
		// Note: ListObjects with delimiter usually doesn't return CommonPrefixes (directories) in Keys
		// Looking at base.go:
		// for _, obj := range page.Contents {
		//     keys = append(keys, *obj.Key)
		//     sizeBytesTot += *obj.Size
		// }
		// So it only returns files.
	})

	t.Run("should list objects recursively", func(t *testing.T) {
		// When
		res, err := client.ListObjects(ctx, "", true)

		// Then
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{
			"dir1/file1.txt",
			"dir1/file2.txt",
			"dir2/file3.txt",
			"file4.txt",
		}, res.Keys)
		assert.Equal(t, int64(1+2+3+4), res.SizeBytesTot)
	})

	t.Run("should list objects with prefix", func(t *testing.T) {
		// When
		res, err := client.ListObjects(ctx, "dir1/", false)

		// Then
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{
			"dir1/file1.txt",
			"dir1/file2.txt",
		}, res.Keys)
	})
}

func TestClient_ListObjectsWithCallback(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()

	s3Client := testutil.SetupS3Client(t, endpoint)
	bucketName := testutil.FakeRandomBucketName()

	testutil.SetupS3Bucket(ctx, t, s3Client, bucketName, []testutil.FakeS3Object{
		{Key: "file1.txt", Body: strings.NewReader("1")},
		{Key: "file2.txt", Body: strings.NewReader("2")},
		{Key: "file3.txt", Body: strings.NewReader("3")},
	})

	conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
	client := s3client.NewAwsClient(conn, func(o *s3.Options) {
		o.Region = "us-east-1"
	})

	t.Run("should call callback for each page", func(t *testing.T) {
		var keys []string
		err := client.ListObjectsWithCallback(ctx, "", true, func(page *s3.ListObjectsV2Output) error {
			for _, obj := range page.Contents {
				keys = append(keys, *obj.Key)
			}
			return nil
		}, func(in any) {
			if listIn, ok := in.(*s3.ListObjectsV2Input); ok {
				listIn.MaxKeys = aws.Int32(1) // Force pagination
			}
		})

		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"file1.txt", "file2.txt", "file3.txt"}, keys)
		assert.Len(t, keys, 3)
	})
}

func TestClient_UploadAndDownload(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()

	s3Client := testutil.SetupS3Client(t, endpoint)
	bucketName := testutil.FakeRandomBucketName()
	key := "large-file.txt"
	content := strings.Repeat("a", 1024*1024) // 1MB

	testutil.SetupS3Bucket(ctx, t, s3Client, bucketName, nil)

	conn := testutil.FakeAwsConnectionWithEndpoint(t, endpoint, bucketName)
	client := s3client.NewAwsClient(conn, func(o *s3.Options) {
		o.Region = "us-east-1"
	})

	t.Run("should upload and download successfully", func(t *testing.T) {
		// When (Upload)
		err := client.Upload(ctx, key, strings.NewReader(content))
		assert.NoError(t, err)

		// Then (Verify Upload)
		testutil.AssertObjectContent(t, s3Client, bucketName, key, content)

		// When (Download)
		tmpFile, err := os.CreateTemp("", "download-test")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		err = client.Download(ctx, key, tmpFile)
		assert.NoError(t, err)

		// Then (Verify Download)
		downloadedContent, err := os.ReadFile(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, content, string(downloadedContent))
	})
}
