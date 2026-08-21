package s3client

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type GrantList []string

func (l GrantList) ToInput() *string {
	if len(l) == 0 {
		return nil
	}
	return aws.String(strings.Join(l, ", "))
}

type Grants struct {
	Read        GrantList
	ReadAcp     GrantList
	WriteAcp    GrantList
	FullControl GrantList
}

type ListObjectsResult struct {
	Keys         []string
	SizeBytesTot int64
}

func (r ListObjectsResult) IsEmpty() bool {
	return len(r.Keys) == 0 || (len(r.Keys) == 1 && strings.HasSuffix(r.Keys[0], "/"))
}

// Range represents a data range as described here: https://www.rfc-editor.org/rfc/rfc9110.html#name-range
type Range struct{}

func MapFromDomainTags(tags []directory.Tag) []s3types.Tag {
	s3Tags := make([]s3types.Tag, len(tags))
	if len(tags) == 0 {
		return s3Tags
	}
	for i, t := range tags {
		s3Tags[i] = s3types.Tag{
			Key:   aws.String(t.Key),
			Value: aws.String(t.Value),
		}
	}
	return s3Tags
}

func MapToDomainTags(s3Tags []s3types.Tag) []directory.Tag {
	tags := make([]directory.Tag, len(s3Tags))
	if len(s3Tags) == 0 {
		return tags
	}
	for i, t := range s3Tags {
		tags[i] = directory.Tag{
			Key:   *t.Key,
			Value: *t.Value,
		}
	}
	return tags
}
