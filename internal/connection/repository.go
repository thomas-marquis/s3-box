package connection

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// ListConnections returns all existing connections
	ListConnections(ctx context.Context) ([]*Connection, error)

	// SaveConnection create or update a connection
	SaveConnection(ctx context.Context, c *Connection) error

	// DeleteConnection deletes given connection
	DeleteConnection(ctx context.Context, id uuid.UUID) error

	// GetConnection returns a connection by name
	GetConnection(ctx context.Context, name string) (*Connection, error)

	SetSelectedConnection(ctx context.Context, id uuid.UUID) error

	GetSelectedConnection(ctx context.Context) (*Connection, error)
}
