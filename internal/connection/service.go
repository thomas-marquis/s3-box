package connection

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ConnectionService interface {
	GetActiveConnectionID(ctx context.Context) (uuid.UUID, error)
}

type connectionServiceImpl struct {
	repository Repository
}

var _ ConnectionService = &connectionServiceImpl{}

func NewConnectionService(repository Repository) *connectionServiceImpl {
	return &connectionServiceImpl{repository}
}

func (s *connectionServiceImpl) GetActiveConnectionID(ctx context.Context) (uuid.UUID, error) {
	conn, err := s.repository.GetSelectedConnection(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error while getting selected connection: %w", err)
	}

	return conn.ID, nil
}
