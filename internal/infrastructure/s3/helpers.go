package s3

import (
	"errors"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"

	"github.com/aws/smithy-go"
)

// isNotFoundError checks if the error is a "not found" error from AWS
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, directory.ErrNotFound) {
		return true
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound"
	}

	return false
}
