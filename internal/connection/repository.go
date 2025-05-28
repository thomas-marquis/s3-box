package connection

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// List returns all existing connections
	List(ctx context.Context) ([]*Connection, error)

	// Save create or update a connection
	Save(ctx context.Context, c *Connection) error

	// Delete deletes given connection
	Delete(ctx context.Context, id uuid.UUID) error

	// GetConnection returns a connection by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Connection, error)

	SetSelected(ctx context.Context, id uuid.UUID) error

	GetSelected(ctx context.Context) (*Connection, error)

	// ExportToJson returns all connections as a JSON byte slice and the count
	ExportToJson(ctx context.Context) (ConnectionExport, error)
}
