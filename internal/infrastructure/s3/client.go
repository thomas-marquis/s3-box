package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client interface {
	ListObjects(ctx context.Context, in *s3.ListObjectsInput) ([]string, error)
	HeadObject(ctx context.Context, in *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObjectAcl(ctx context.Context, in *s3.GetObjectAclInput, opts ...func(*s3.Options)) (*s3.GetObjectAclOutput, error)
	CopyObject(ctx context.Context, in *s3.CopyObjectInput, opts ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, opts ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}
