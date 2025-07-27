package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/dto"
	"io"
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

func (r *FyneConnectionsRepository) Get(_ context.Context) (*connection_deck.Deck, error) {
	dtos, err := r.loadConnectionsDTO()
	if err != nil {
		return nil, fmt.Errorf("load connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	return dtos.ToConnections(), nil
}

func (r *FyneConnectionsRepository) Save(_ context.Context, deck *connection_deck.Deck) error {
	dtos := dto.NewConnectionsDTO(deck)
	jsonContent, err := dtos.Serialize()
	if err != nil {
		return fmt.Errorf("serialize connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}
	r.prefs.SetString(allConnectionsKey, string(jsonContent))
	return nil
}

func (r *FyneConnectionsRepository) Export(_ context.Context, file io.Writer) error {
	deck, err := r.Get(context.Background())
	if err != nil {
		return fmt.Errorf("get connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	dtos := dto.NewConnectionsDTO(deck)
	jsonContent, err := dtos.Serialize()
	if err != nil {
		return fmt.Errorf("serialize connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	if _, err = file.Write(jsonContent); err != nil {
		return fmt.Errorf("write connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}
	return nil
}

func (r *FyneConnectionsRepository) loadConnectionsDTO() (*dto.ConnectionsDTO, error) {
	content := r.prefs.String(allConnectionsKey)
	if content == "" || content == "null" {
		return &dto.ConnectionsDTO{}, nil
	}

	return dto.NewConnectionsDTOFromJSON([]byte(content))
}
