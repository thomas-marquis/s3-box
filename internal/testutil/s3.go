package testutil

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	awsHttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

type FakeS3Object struct {
	Key  string
	Body io.Reader
}

func SetupS3testContainer(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()

	lsContainer, err := localstack.Run(ctx, "localstack/localstack:3.0")
	require.NoError(t, err)

	endpoint, err := lsContainer.PortEndpoint(ctx, "4566", "")
	require.NoError(t, err)

	return "http://" + endpoint, func() {
		_ = lsContainer.Terminate(ctx)
	}
}

func SetupS3Client(t *testing.T, endpoint string) *s3.Client {
	t.Helper()

	awsCfg := aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(endpoint),
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return s3Client
}

func SetupS3Bucket(ctx context.Context, t *testing.T, client *s3.Client, bucketName string, content []FakeS3Object) {
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

func ListKeys(t *testing.T, client *s3.Client, bucketName, prefix string) []string {
	t.Helper()

	res, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	})
	require.NoError(t, err)

	var keys []string
	for _, obj := range res.Contents {
		k := *obj.Key
		if k == prefix {
			continue
		}
		keys = append(keys, k)
	}

	return keys
}

func GetObject(t *testing.T, client *s3.Client, bucketName, key string) io.ReadCloser {
	t.Helper()

	res, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	require.NoError(t, err)

	return res.Body
}

func PutObject(t *testing.T, client *s3.Client, bucketName, key string, body io.Reader) {
	t.Helper()

	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   body,
	})
	require.NoError(t, err)
}

func AssertObjectContent(t *testing.T, client *s3.Client, bucketName, key, expectedContent string) bool {
	t.Helper()

	body := GetObject(t, client, bucketName, key)
	defer body.Close() //nolint:errcheck

	content, err := io.ReadAll(body)

	return assert.NoError(t, err) &&
		assert.Equal(t, expectedContent, string(content))
}

func AssertJSONObjectContent(t *testing.T, client *s3.Client, bucketName, key, expectedJSONContent string) bool {
	t.Helper()

	body := GetObject(t, client, bucketName, key)
	defer body.Close() //nolint:errcheck

	content, err := io.ReadAll(body)

	return assert.NoError(t, err) &&
		assert.JSONEq(t, expectedJSONContent, string(content))
}

func AssertObjectNotExists(t *testing.T, client *s3.Client, bucketName, key string) bool {
	t.Helper()

	_, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	var opErr *smithy.OperationError
	if !assert.ErrorAs(t, err, &opErr) {
		return false
	}
	var respErr *awsHttp.ResponseError
	if !assert.ErrorAs(t, opErr.Err, &respErr) {
		return false
	}
	return assert.Equal(t, http.StatusNotFound, respErr.Response.StatusCode)
}
