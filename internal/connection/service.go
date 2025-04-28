package connection

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ConnectionService interface {
	GetActiveConnectionID(ctx context.Context) (uuid.UUID, error)
}

type ConnectionServiceImpl struct {
	repository Repository
}

var _ ConnectionService = &ConnectionServiceImpl{}

func NewConnectionService(repository Repository) *ConnectionServiceImpl {
	return &ConnectionServiceImpl{repository}
}

func (s *ConnectionServiceImpl) GetActiveConnectionID(ctx context.Context) (uuid.UUID, error) {
	conn, err := s.repository.GetSelectedConnection(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error while getting selected connection: %w", err)
	}

	return conn.ID, nil
}
