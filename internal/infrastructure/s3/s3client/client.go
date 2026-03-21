package s3client

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BaseAPI interface {
	PutObject(ctx context.Context, key string, body io.Reader, opts ...Option) error
	GetObjectGrants(ctx context.Context, key string, opts ...Option) (Grants, error)
	DeleteObject(ctx context.Context, key string, opts ...Option) error
	GetObject(ctx context.Context, key string, opts ...Option) (*s3.GetObjectOutput, error)
	ListObjects(ctx context.Context, prefix string, recursive bool, opts ...Option) (ListObjectsResult, error)
	ListObjectsWithCallback(ctx context.Context, prefix string, recursive bool, callback func(page *s3.ListObjectsV2Output) error, opts ...Option) error
	Download(ctx context.Context, key string, writer io.WriterAt, opts ...Option) error
	Upload(ctx context.Context, key string, body io.Reader, opts ...Option) error
}

type Client interface {
	BaseAPI

	RenameObject(ctx context.Context, oldKey, newKey string, opts ...Option) error
}

type clientImpl struct {
	api BaseAPI

	client     *s3.Client
	bucket     string
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

func newClientImpl(client *s3.Client, bucket string, api BaseAPI) *clientImpl {
	return &clientImpl{
		api:        api,
		client:     client,
		bucket:     bucket,
		downloader: manager.NewDownloader(client),
		uploader:   manager.NewUploader(client),
	}
}

func (c *clientImpl) RenameObject(ctx context.Context, oldKey, newKey string, opts ...Option) error {
	hIn := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(oldKey),
	}
	for _, opt := range opts {
		opt(hIn)
	}
	headRes, err := c.client.HeadObject(ctx, hIn)
	if err != nil {
		return err
	}

	grants, err := c.api.GetObjectGrants(ctx, oldKey, opts...)
	if err != nil {
		return err
	}

	cpyInput := &s3.CopyObjectInput{
		Bucket:                         aws.String(c.bucket),
		CopySource:                     aws.String(c.bucket + "/" + oldKey),
		Key:                            aws.String(newKey),
		CacheControl:                   headRes.CacheControl,
		ContentDisposition:             headRes.ContentDisposition,
		ContentEncoding:                headRes.ContentEncoding,
		ContentLanguage:                headRes.ContentLanguage,
		ContentType:                    headRes.ContentType,
		CopySourceSSECustomerAlgorithm: headRes.SSECustomerAlgorithm,
		CopySourceSSECustomerKeyMD5:    headRes.SSECustomerKeyMD5,
		GrantFullControl:               grants.FullControl.ToInput(),
		GrantRead:                      grants.Read.ToInput(),
		GrantReadACP:                   grants.ReadAcp.ToInput(),
		GrantWriteACP:                  grants.WriteAcp.ToInput(),
		Metadata:                       headRes.Metadata,
		MetadataDirective:              "REPLACE",
		ObjectLockLegalHoldStatus:      headRes.ObjectLockLegalHoldStatus,
		ObjectLockMode:                 headRes.ObjectLockMode,
		ObjectLockRetainUntilDate:      headRes.ObjectLockRetainUntilDate,
		SSECustomerAlgorithm:           headRes.SSECustomerAlgorithm,
		SSECustomerKeyMD5:              headRes.SSECustomerKeyMD5,
		SSEKMSKeyId:                    headRes.SSEKMSKeyId,
		ServerSideEncryption:           headRes.ServerSideEncryption,
		StorageClass:                   headRes.StorageClass,
		TaggingDirective:               "COPY",
		WebsiteRedirectLocation:        headRes.WebsiteRedirectLocation,
	}
	for _, opt := range opts {
		opt(cpyInput)
	}
	if _, err := c.client.CopyObject(ctx, cpyInput); err != nil {
		return err
	}

	return c.api.DeleteObject(ctx, oldKey, opts...)
}

func (c *clientImpl) PutObject(ctx context.Context, key string, body io.Reader, opts ...Option) error {
	return c.api.PutObject(ctx, key, body, opts...)
}

func (c *clientImpl) GetObjectGrants(ctx context.Context, key string, opts ...Option) (Grants, error) {
	return c.api.GetObjectGrants(ctx, key, opts...)
}

func (c *clientImpl) DeleteObject(ctx context.Context, key string, opts ...Option) error {
	return c.api.DeleteObject(ctx, key, opts...)
}

func (c *clientImpl) GetObject(ctx context.Context, key string, opts ...Option) (*s3.GetObjectOutput, error) {
	return c.api.GetObject(ctx, key, opts...)
}

func (c *clientImpl) ListObjects(ctx context.Context, prefix string, recursive bool, opts ...Option) (ListObjectsResult, error) {
	return c.api.ListObjects(ctx, prefix, recursive, opts...)
}

func (c *clientImpl) ListObjectsWithCallback(ctx context.Context, prefix string, recursive bool, callback func(page *s3.ListObjectsV2Output) error, opts ...Option) error {
	return c.api.ListObjectsWithCallback(ctx, prefix, recursive, callback, opts...)
}

func (c *clientImpl) Download(ctx context.Context, key string, writer io.WriterAt, opts ...Option) error {
	return c.api.Download(ctx, key, writer, opts...)
}

func (c *clientImpl) Upload(ctx context.Context, key string, body io.Reader, opts ...Option) error {
	return c.api.Upload(ctx, key, body, opts...)
}

type Option func(any)

func WithContentLength(length int64) Option {
	return func(in any) {
		switch i := in.(type) {
		case *s3.PutObjectInput:
			i.ContentLength = aws.Int64(length)
		}
	}
}
