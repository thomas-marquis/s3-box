package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/dto"
)

const (
	allConnectionsKey = "allConnections"
)

type FyneConnectionsRepository struct {
	prefs fyne.Preferences
	bus   event.Bus
}

var _ connection_deck.Repository = &FyneConnectionsRepository{}

func NewFyneConnectionsRepository(
	prefs fyne.Preferences,
	bus event.Bus,
) *FyneConnectionsRepository {
	r := &FyneConnectionsRepository{prefs: prefs, bus: bus}

	bus.SubscribeV2().
		On(event.Is(connection_deck.SelectEventType), r.handleSelect).
		On(event.Is(connection_deck.CreateEventType), r.handleCreate).
		On(event.Is(connection_deck.RemoveEventType), r.handleRemove).
		On(event.Is(connection_deck.UpdateEventType), r.handleUpdate).
		ListenWithWorkers(1)

	return r
}

func (r *FyneConnectionsRepository) Get(_ context.Context) (*connection_deck.Deck, error) {
	dtos, err := r.loadConnectionsDTO()
	if err != nil {
		return nil, fmt.Errorf("load connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	return dtos.ToConnections(), nil
}

func (r *FyneConnectionsRepository) Export(_ context.Context, file io.Writer) error {
	deck, err := r.Get(context.Background())
	if err != nil {
		return fmt.Errorf("get connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	dtos := dto.NewConnectionsDTO(deck)
	jsonContent, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("serialize connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	if _, err = file.Write(jsonContent); err != nil {
		return fmt.Errorf("write connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}
	return nil
}

func (r *FyneConnectionsRepository) saveDeck(_ context.Context, deck *connection_deck.Deck) error {
	dtos := dto.NewConnectionsDTO(deck)
	jsonContent, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("serialize connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}
	r.prefs.SetString(allConnectionsKey, string(jsonContent))
	return nil
}

func (r *FyneConnectionsRepository) loadConnectionsDTO() (*dto.ConnectionsDTO, error) {
	content := r.prefs.String(allConnectionsKey)
	if content == "" || content == "null" {
		return &dto.ConnectionsDTO{}, nil
	}

	return dto.NewConnectionsDTOFromJSON([]byte(content))
}

func (r *FyneConnectionsRepository) handleSelect(evt event.Event) {
	ctx := evt.Context()
	e := evt.(connection_deck.SelectEvent)
	if err := r.saveDeck(ctx, e.Deck()); err != nil {
		r.bus.PublishV2(connection_deck.NewSelectFailureEvent(err, e.Connection()))
	}
	r.bus.PublishV2(connection_deck.NewSelectSuccessEvent(e.Deck(), e.Connection()))
}

func (r *FyneConnectionsRepository) handleCreate(evt event.Event) {
	ctx := evt.Context()
	e := evt.(connection_deck.CreateEvent)
	if err := r.saveDeck(ctx, e.Deck()); err != nil {
		r.bus.PublishV2(connection_deck.NewCreateFailureEvent(err, e.Connection()))
	}
	r.bus.PublishV2(connection_deck.NewCreateSuccessEvent(e.Deck(), e.Connection()))
}

func (r *FyneConnectionsRepository) handleRemove(evt event.Event) {
	ctx := evt.Context()
	e := evt.(connection_deck.RemoveEvent)
	if err := r.saveDeck(ctx, e.Deck()); err != nil {
		r.bus.PublishV2(connection_deck.NewRemoveFailureEvent(err, e.RemovedIndex(), e.WasSelected(), e.Connection()))
	}
	r.bus.PublishV2(connection_deck.NewRemoveSuccessEvent(e.Deck(), e.Connection()))
}

func (r *FyneConnectionsRepository) handleUpdate(evt event.Event) {
	ctx := evt.Context()
	e := evt.(connection_deck.UpdateEvent)
	if err := r.saveDeck(ctx, e.Deck()); err != nil {
		r.bus.PublishV2(connection_deck.NewUpdateFailureEvent(err, e.Previous()))
	}
	r.bus.PublishV2(connection_deck.NewUpdateSuccessEvent(e.Deck(), e.Connection()))
}
