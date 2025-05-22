package connection

import (
	"strings"

	"github.com/google/uuid"
)

type ConnectionType string

func (c ConnectionType) String() string {
	return string(c)
}

const (
	AWSConnectionType     ConnectionType = "aws"
	S3LikeConnectionType  ConnectionType = "s3-like"
	DefaultConnectionType ConnectionType = S3LikeConnectionType
)

func NewConnectionTypeFromString(s string) ConnectionType {
	switch strings.ToLower(s) {
	case AWSConnectionType.String():
		return AWSConnectionType
	case S3LikeConnectionType.String():
		return S3LikeConnectionType
	default:
		return DefaultConnectionType
	}
}

type ConnectionOption func(*Connection)

func AsAWSConnection(region string) ConnectionOption {
	return func(c *Connection) {
		c.Type = AWSConnectionType
		c.Region = region
		c.UseTls = true
		c.Server = ""
	}
}

func AsS3LikeConnection(server string, useTLS bool) ConnectionOption {
	return func(c *Connection) {
		c.Type = S3LikeConnectionType
		c.Server = server
		c.UseTls = useTLS
		c.Region = ""
	}
}

func WithReadOnlyOption(readOnly bool) ConnectionOption {
	return func(c *Connection) {
		c.ReadOnly = readOnly
	}
}

type Connection struct {
	ID         uuid.UUID
	Name       string
	Server     string
	SecretKey  string
	AccessKey  string
	BucketName string
	UseTls     bool
	IsSelected bool
	Region     string
	Type       ConnectionType
	ReadOnly   bool
}

func NewConnection(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	c := &Connection{
		ID:         uuid.New(),
		Name:       name,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		BucketName: bucket,
		ReadOnly:   false,
	}

	for _, opt := range options {
		opt(c)
	}

	if c.Type == "" {
		AsAWSConnection("us-east-1")(c)
	}

	return c
}

func NewEmptyConnection() *Connection {
	return NewConnection("", "", "", "", AsAWSConnection("us-east-1"))
}

func (c *Connection) Update(other *Connection) {
	c.Name = other.Name
	c.Server = other.Server
	c.SecretKey = other.SecretKey
	c.AccessKey = other.AccessKey
	c.BucketName = other.BucketName
	c.UseTls = other.UseTls
	c.Region = other.Region
	c.Type = other.Type
	c.ReadOnly = other.ReadOnly
}

type ConnectionExport struct {
	JSONData []byte
	Count    int
}
