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

	// GetConnection returns a connection by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Connection, error)

	SetSelectedConnection(ctx context.Context, id uuid.UUID) error

	GetSelectedConnection(ctx context.Context) (*Connection, error)

	// ExportToJson returns all connections as a JSON byte slice
	ExportToJson(ctx context.Context) ([]byte, error)
}
