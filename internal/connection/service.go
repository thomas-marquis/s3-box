package connection

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ConnectionService interface {
	GetActiveConnectionID(ctx context.Context) (uuid.UUID, error) // TODO

	// Select a connection by its ID.
	// Return an error if the connection ID does not exist or if the selection fails.
	Select(ctx context.Context, ID uuid.UUID) error
}

type connectionServiceImpl struct {
	repository Repository
}

var _ ConnectionService = &connectionServiceImpl{}

func NewConnectionService(repository Repository) *connectionServiceImpl {
	return &connectionServiceImpl{repository}
}

func (s *connectionServiceImpl) GetActiveConnectionID(ctx context.Context) (uuid.UUID, error) {
	conn, err := s.repository.GetSelected(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error while getting selected connection: %w", err)
	}

	return conn.ID(), nil
}

func (s *connectionServiceImpl) Select(ctx context.Context, ID uuid.UUID) error {
	allConns, err := s.repository.List(ctx)
	if err != nil {
		return err
	}
	for _, c := range allConns {
		if c.ID() == ID {
			c.Select()
			if err := s.repository.Save(ctx, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
