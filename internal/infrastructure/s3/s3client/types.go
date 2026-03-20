package s3client

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
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
