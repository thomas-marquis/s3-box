package connection

import "github.com/google/uuid"

type Connection struct {
	ID         uuid.UUID
	Name       string
	Server     string
	SecretKey  string
	AccessKey  string
	BucketName string
	UseTls     bool
	IsSelected bool
}

func NewConnection(name, server, accessKey, secretKey, bucket string, useTLS bool) *Connection {
	return &Connection{
		ID:         uuid.New(),
		Name:       name,
		Server:     server,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		BucketName: bucket,
		UseTls:     useTLS,
	}
}
