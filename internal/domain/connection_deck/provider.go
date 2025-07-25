package connection_deck

import "strings"

type Provider string

func (c Provider) String() string {
	return string(c)
}

const (
	nilProvider     Provider = ""
	AWSProvider     Provider = "aws"
	S3LikeProvider  Provider = "s3-like"
	DefaultProvider Provider = S3LikeProvider
)

func NewProviderFromString(s string) Provider {
	switch strings.ToLower(s) {
	case AWSProvider.String():
		return AWSProvider
	case S3LikeProvider.String():
		return S3LikeProvider
	default:
		return DefaultProvider
	}
}
