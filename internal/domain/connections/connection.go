package connections

import (
	"github.com/google/uuid"
)

type ConnectionID uuid.UUID

var nilConnectionID ConnectionID = ConnectionID(uuid.Nil)

func NewConnectionID() ConnectionID {
	return ConnectionID(uuid.New())
}

func (id ConnectionID) String() string {
	return uuid.UUID(id).String()
}

func (id ConnectionID) Is(conn *Connection) bool {
	if conn == nil {
		return false
	}
	return id == conn.id
}

type Connection struct {
	id        ConnectionID
	name      string
	accessKey string
	secretKey string
	bucket    string
	server    string
	region    string
	useTLS    bool
	readOnly  bool
	revision  int
	provider  Provider
}

func newConnection(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	conn := &Connection{
		id:        NewConnectionID(),
		name:      name,
		accessKey: accessKey,
		secretKey: secretKey,
		bucket:    bucket,
		readOnly:  false,
		provider:  nilProvider,
	}
	for _, opt := range options {
		opt(conn)
	}

	if conn.provider == nilProvider {
		AsAWS("us-east-1")(conn)
	}
	return conn
}

func (c *Connection) Is(other *Connection) bool {
	if other == nil {
		return false
	}
	return c.id == other.id
}

func (c *Connection) ID() ConnectionID {
	return c.id
}

func (c *Connection) Name() string {
	return c.name
}

func (c *Connection) SetName(name string) {
	if name != c.name {
		c.revision++
		c.name = name
	}
}

func (c *Connection) AccessKey() string {
	return c.accessKey
}

func (c *Connection) SetAccessKey(accessKey string) {
	if accessKey != c.accessKey {
		c.revision++
		c.accessKey = accessKey
	}
}

func (c *Connection) SecretKey() string {
	return c.secretKey
}

func (c *Connection) SetSecretKey(secretKey string) {
	if secretKey != c.secretKey {
		c.revision++
		c.secretKey = secretKey
	}
}

func (c *Connection) Bucket() string {
	return c.bucket
}

func (c *Connection) SetBucket(bucket string) {
	if bucket != c.bucket {
		c.revision++
		c.bucket = bucket
	}
}

func (c *Connection) Server() string {
	return c.server
}

func (c *Connection) SetServer(server string) {
	if server != c.server {
		c.revision++
		c.server = server
	}
}

func (c *Connection) Region() string {
	return c.region
}

func (c *Connection) SetRegion(region string) {
	if region == c.region || c.provider != AWSProvider {
		return
	}
	c.revision++
	c.region = region
}

func (c *Connection) UseTLS() bool {
	return c.useTLS
}

func (c *Connection) SetUseTLS(useTLS bool) {
	if useTLS == c.useTLS || c.provider != S3LikeProvider {
		return
	}
	c.revision++
	c.useTLS = useTLS
}

// Revision returns the current revision number of the connection.
// This number is incremented each time the connection is updated.
func (c *Connection) Revision() int {
	return c.revision
}

func (c *Connection) ReadOnly() bool {
	return c.readOnly
}

func (c *Connection) SetReadOnly(readOnly bool) {
	if readOnly != c.readOnly {
		c.revision++
		c.readOnly = readOnly
	}
}

func (c *Connection) Provider() Provider {
	return c.provider
}

func (c *Connection) ASAWS(region string) {
	if c.provider == AWSProvider {
		return
	}
	c.revision++
	AsAWS(region)(c)
}

func (c *Connection) ASS3Like(server string, useTLS bool) {
	if c.provider == S3LikeProvider {
		return
	}
	c.revision++
	AsS3Like(server, useTLS)(c)
}
