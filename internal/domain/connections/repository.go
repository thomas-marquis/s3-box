package connections

import "context"

type Repository interface {
	Get(ctx context.Context) (*Connections, error)
	Export(ctx context.Context) ([]byte, error)
}
