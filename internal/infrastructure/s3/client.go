package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Provider interface {
	RenameObject(ctx context.Context, client *s3.Client, oldKey, newKey string) error
}
