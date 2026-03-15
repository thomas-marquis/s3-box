package s3client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Grants struct {
	Read        []string
	ReadAcp     []string
	WriteAcp    []string
	FullControl []string
}

type Client interface {
	ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, opts ...func(*s3.Options))
	HeadObject(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options))
	PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options))
	GetObject(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options))
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, opts ...func(*s3.Options))
	GetObjectGrants(ctx context.Context, input *s3.GetObjectAclInput, opts ...func(*s3.Options)) (Grants, error)
}

type baseClient struct{}
