package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/dto"
)

const (
	allConnectionsKey = "allConnections"
)

type FyneConnectionsRepository struct {
	prefs fyne.Preferences
}

var _ connection_deck.Repository = &FyneConnectionsRepository{}

func NewFyneConnectionsRepository(prefs fyne.Preferences) *FyneConnectionsRepository {
	return &FyneConnectionsRepository{prefs: prefs}
}

func (r *FyneConnectionsRepository) Get(ctx context.Context) (*connection_deck.Deck, error) {
	dtos, err := r.loadConnectionsDTO()
	if err != nil {
		return nil, fmt.Errorf("load connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	return dtos.ToConnections(), nil
}

func (r *FyneConnectionsRepository) Save(ctx context.Context, conn *connection_deck.Deck) error {
	dtos := dto.NewConnectionsDTO(conn)
	jsonContent, err := dtos.Serialize()
	if err != nil {
		return fmt.Errorf("serialize connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}
	r.prefs.SetString(allConnectionsKey, string(jsonContent))
	return nil
}

func (r *FyneConnectionsRepository) GetByID(ctx context.Context, id connection_deck.ConnectionID) (*connection_deck.Connection, error) {
	conns, err := r.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get by id, fail to get connections %s: %w", id, err)
	}
	c, err := conns.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get connection %s: %w", id, err)
	}

	return c, nil
}

func (r *FyneConnectionsRepository) loadConnectionsDTO() (*dto.ConnectionsDTO, error) {
	content := r.prefs.String(allConnectionsKey)
	if content == "" || content == "null" {
		return &dto.ConnectionsDTO{}, nil
	}

	return dto.NewConnectionsDTOFromJSON([]byte(content))
}
