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
		if region == "" {
			return
		}
		c.Type = AWSConnectionType
		c.Region = region
		c.UseTls = true
		c.Server = ""
	}
}

func AsS3LikeConnection(server string, useTLS bool) ConnectionOption {
	return func(c *Connection) {
		if server == "" {
			return
		}
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

func WithRevision(revision int) ConnectionOption {
	return func(c *Connection) {
		c.revision = revision
	}
}

func WithSelected(selected bool) ConnectionOption {
	return func(c *Connection) {
		c.selected = selected
	}
}

func WithUseTLS(useTLS bool) ConnectionOption {
	return func(c *Connection) {
		c.UseTls = useTLS
	}
}

func WithID(id uuid.UUID) ConnectionOption {
	return func(c *Connection) {
		c.id = id
	}
}

type Connection struct {
	Name       string
	Server     string
	SecretKey  string
	AccessKey  string
	BucketName string
	UseTls     bool
	Region     string
	Type       ConnectionType
	ReadOnly   bool

	id       uuid.UUID
	selected bool
	revision int
}

func NewConnection(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	c := &Connection{
		id:         uuid.New(),
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

func NewEmptyConnection(options ...ConnectionOption) *Connection {
	return NewConnection("", "", "", "", options...)
}

func (c *Connection) ID() uuid.UUID {
	return c.id
}

func (c *Connection) Is(other *Connection) bool {
	if other == nil {
		return false
	}
	return c.id == other.id
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
	c.IncRevision()
}

// Compare checks if the current connection is equal to another connection.
func (c *Connection) Compare(other *Connection) bool {
	return c.Name == other.Name &&
		c.Server == other.Server &&
		c.SecretKey == other.SecretKey &&
		c.AccessKey == other.AccessKey &&
		c.BucketName == other.BucketName &&
		c.UseTls == other.UseTls &&
		c.Region == other.Region &&
		c.Type == other.Type &&
		c.ReadOnly == other.ReadOnly &&
		c.revision == other.revision
}

// Revision returns the current revision number of the connection.
// This number is incremented each time the connection is updated.
func (c *Connection) Revision() int {
	return c.revision
}

// SetRevision sets the revision number of the connection only if no version has been set yet.
// It's useful for initializing the revision number when the connection is first created.
func (c *Connection) SetRevision(newRevision int) {
	if c.revision > 0 {
		c.revision = newRevision
	}
}

// IncRevision increments the revision number of the connection.
func (c *Connection) IncRevision() {
	c.revision++
}

func (c *Connection) Select() {
	c.selected = true
}

func (c *Connection) Selected() bool {
	return c.selected
}

func (c *Connection) Unselect() {
	c.selected = false
}

type ConnectionExport struct {
	JSONData []byte
	Count    int
}
