package s3client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type baseApiImpl struct {
	client     *s3.Client
	bucket     string
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

var _ BaseAPI = (*baseApiImpl)(nil)

func newBaseApiImpl(client *s3.Client, bucket string) *baseApiImpl {
	return &baseApiImpl{
		client:     client,
		bucket:     bucket,
		downloader: manager.NewDownloader(client),
		uploader:   manager.NewUploader(client),
	}
}

func (c *baseApiImpl) PutObject(ctx context.Context, key string, body io.Reader, opts ...Option) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	for _, opt := range opts {
		opt(in)
	}
	_, err := c.client.PutObject(ctx, in)
	return c.handleS3SdkError(err, key)
}

func (c *baseApiImpl) DeleteObject(ctx context.Context, key string, opts ...Option) error {
	in := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	for _, opt := range opts {
		opt(in)
	}
	_, err := c.client.DeleteObject(ctx, in)
	return c.handleS3SdkError(err, key)
}

func (c *baseApiImpl) GetObject(ctx context.Context, key string, opts ...Option) (*s3.GetObjectOutput, error) {
	in := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	for _, opt := range opts {
		opt(in)
	}
	res, err := c.client.GetObject(ctx, in)
	return res, c.handleS3SdkError(err, key)
}

func (c *baseApiImpl) ListObjects(ctx context.Context, prefix string, recursive bool, opts ...Option) (ListObjectsResult, error) {
	var keys []string
	var sizeBytesTot int64

	if err := c.ListObjectsWithCallback(ctx, prefix, recursive, func(page *s3.ListObjectsV2Output) error {
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
			sizeBytesTot += *obj.Size
		}
		return nil
	}, opts...); err != nil {
		return ListObjectsResult{}, err
	}

	return ListObjectsResult{
		Keys:         keys,
		SizeBytesTot: sizeBytesTot,
	}, nil
}

func (c *baseApiImpl) ListObjectsWithCallback(ctx context.Context, prefix string, recursive bool, callback func(page *s3.ListObjectsV2Output) error, opts ...Option) error {
	var delimiter *string
	if !recursive {
		delimiter = aws.String("/")
	}

	inputs := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: delimiter,
		MaxKeys:   aws.Int32(1000),
	}
	for _, opt := range opts {
		opt(inputs)
	}
	paginator := s3.NewListObjectsV2Paginator(c.client, inputs)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return c.handleS3SdkError(err, prefix)
		}

		if err := callback(page); err != nil {
			return err
		}
	}

	return nil
}

func (c *baseApiImpl) GetObjectGrants(ctx context.Context, key string, opts ...Option) (Grants, error) {
	return Grants{}, nil
}

func (c *baseApiImpl) Download(ctx context.Context, key string, writer io.WriterAt, opts ...Option) error {
	in := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	for _, opt := range opts {
		opt(in)
	}
	_, err := c.downloader.Download(ctx, writer, in)
	return c.handleS3SdkError(err, key)
}

func (c *baseApiImpl) Upload(ctx context.Context, key string, body io.Reader, opts ...Option) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	for _, opt := range opts {
		opt(in)
	}
	_, err := c.uploader.Upload(ctx, in)
	return c.handleS3SdkError(err, key)
}

func (c *baseApiImpl) handleS3SdkError(err error, objName string) error {
	if err == nil {
		return nil
	}

	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		return errors.Join(
			directory.ErrNotFound,
			fmt.Errorf("object %s not found in bucket %s: %w",
				objName, c.bucket, err),
		)
	}

	var nsb *s3types.NoSuchBucket
	if errors.As(err, &nsb) {
		return errors.Join(
			directory.ErrNotFound,
			fmt.Errorf("bucket %s not found: %w", c.bucket, err),
		)
	}

	return fmt.Errorf("another kind of s3 error occurred: %w", err)
}
