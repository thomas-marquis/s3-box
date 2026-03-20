package s3client

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func TestBaseApiImpl_handleS3SdkError(t *testing.T) {
	c := &baseApiImpl{bucket: "test-bucket"}

	t.Run("should return nil if err is nil", func(t *testing.T) {
		err := c.handleS3SdkError(nil, "key")
		assert.NoError(t, err)
	})

	t.Run("should wrap NoSuchKey with directory.ErrNotFound", func(t *testing.T) {
		s3Err := &types.NoSuchKey{}
		err := c.handleS3SdkError(s3Err, "key")
		assert.ErrorIs(t, err, directory.ErrNotFound)
		assert.Contains(t, err.Error(), "object key not found")
	})

	t.Run("should wrap NoSuchBucket with directory.ErrNotFound", func(t *testing.T) {
		s3Err := &types.NoSuchBucket{}
		err := c.handleS3SdkError(s3Err, "key")
		assert.ErrorIs(t, err, directory.ErrNotFound)
		assert.Contains(t, err.Error(), "bucket test-bucket not found")
	})

	t.Run("should return generic error for other errors", func(t *testing.T) {
		otherErr := errors.New("something else")
		err := c.handleS3SdkError(otherErr, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "another kind of s3 error occurred")
		assert.Contains(t, err.Error(), "something else")
	})
}
