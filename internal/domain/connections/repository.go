package connections

import "context"

type Repository interface {
	Get(ctx context.Context) (*Connections, error)
	Save(ctx context.Context, conn *Connections) error
}
