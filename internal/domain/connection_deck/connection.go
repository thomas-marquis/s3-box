package connection_deck

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

// Is return true if the connection is EXACTLY the same as the provided one.
func (c *Connection) Is(other *Connection) bool {
	if other == nil {
		return false
	}
	return c.id == other.id && c.revision == other.revision
}

func (c *Connection) ID() ConnectionID {
	return c.id
}

func (c *Connection) Name() string {
	return c.name
}

func (c *Connection) Rename(name string) {
	if name != c.name && !c.readOnly {
		c.revision++
		c.name = name
	}
}

func (c *Connection) AccessKey() string {
	return c.accessKey
}

func (c *Connection) UpdateAccessKey(accessKey string) {
	if accessKey != c.accessKey && !c.readOnly {
		c.revision++
		c.accessKey = accessKey
	}
}

func (c *Connection) SecretKey() string {
	return c.secretKey
}

func (c *Connection) UpdateSecretKey(secretKey string) {
	if secretKey != c.secretKey && !c.readOnly {
		c.revision++
		c.secretKey = secretKey
	}
}

func (c *Connection) Bucket() string {
	return c.bucket
}

func (c *Connection) UpdateBucket(bucket string) {
	if bucket != c.bucket && !c.readOnly {
		c.revision++
		c.bucket = bucket
	}
}

func (c *Connection) Server() string {
	return c.server
}

func (c *Connection) UpdateServer(server string) {
	if server != c.server && !c.readOnly {
		c.revision++
		c.server = server
	}
}

func (c *Connection) Region() string {
	return c.region
}

func (c *Connection) ChangeRegion(region string) {
	if region == c.region || c.provider != AWSProvider || c.readOnly {
		return
	}
	c.revision++
	c.region = region
}

func (c *Connection) IsTLSActivated() bool {
	return c.useTLS
}

func (c *Connection) TurnTLSOn() {
	if !c.useTLS || c.provider != S3LikeProvider || c.readOnly {
		return
	}
	c.revision++
	c.useTLS = true
}

func (c *Connection) TurnTLSOff() {
	if c.useTLS || c.provider != S3LikeProvider || c.readOnly {
		return
	}
	c.revision++
	c.useTLS = false
}

// Revision returns the current revision number of the connection.
// This number is incremented each time the connection is updated.
func (c *Connection) Revision() int {
	return c.revision
}

func (c *Connection) ReadOnly() bool {
	return c.readOnly
}

// SetReadOnly updates the read-only state of the connection.
// Once read-only mode enabled, all other setters will be inoperant
func (c *Connection) SetReadOnly(readOnly bool) {
	if readOnly != c.readOnly {
		c.revision++
		c.readOnly = readOnly
	}
}

func (c *Connection) Provider() Provider {
	return c.provider
}

func (c *Connection) AsAWS(region string) {
	if c.provider == AWSProvider || c.readOnly {
		return
	}
	c.revision++
	AsAWS(region)(c)
}

func (c *Connection) AsS3Like(server string, useTLS bool) {
	if c.provider == S3LikeProvider || c.readOnly {
		return
	}
	c.revision++
	AsS3Like(server, useTLS)(c)
}
