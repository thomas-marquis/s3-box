package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/s3box"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/dto"
	"github.com/thomas-marquis/s3-box/internal/u"
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

	bus.Subscribe().
		On(event.Is(connection_deck.SelectConnectionTriggeredType), r.handleSelect).
		On(event.Is(connection_deck.CreateConnectionTriggeredType), r.handleCreate).
		On(event.Is(connection_deck.RemoveConnectionTriggeredType), r.handleRemove).
		On(event.Is(connection_deck.UpdateConnectionTriggeredType), r.handleUpdate).
		On(event.Is(connection_deck.ImportTriggeredType), r.handleImportTriggered).
		On(event.Is(s3box.UserValidationAcceptedType), r.handleUserValidationAccepted).
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
	jsonContent, err := json.MarshalIndent(dtos, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}

	if _, err = file.Write(jsonContent); err != nil {
		return fmt.Errorf("write connections: %w", errors.Join(err, connection_deck.ErrTechnical))
	}
	return nil
}

func (r *FyneConnectionsRepository) handleImportTriggered(evt event.Event) {
	pl := evt.Payload().(connection_deck.ImportTriggered)
	defer u.SkipD(pl.JSONFile.Close)

	handleErr := func(err error) {
		r.bus.Publish(evt.NewFollowup(connection_deck.ImportFailed{
			Deck: pl.Deck,
			Err:  err,
		}))
	}

	content, err := io.ReadAll(pl.JSONFile)
	if err != nil {
		handleErr(fmt.Errorf("read connections from file: %w", errors.Join(err, connection_deck.ErrTechnical)))
		return
	}

	dtos, err := dto.NewConnectionsDTOFromJSON(content)
	if err != nil {
		handleErr(fmt.Errorf("invalid JSON: %w", errors.Join(err, connection_deck.ErrTechnical)))
		return
	}

	newConns := dtos.ToConnections().Get()

	nextEvt := evt.NewFollowup(connection_deck.ImportConfirmationAsked{
		Deck:           pl.Deck,
		NewConnections: newConns,
	})

	msg := strings.Builder{}
	u.SkipV(fmt.Fprintf(&msg, "%d connections will be imported:\n", len(newConns)))
	for _, conn := range newConns {
		u.SkipV(fmt.Fprintf(&msg, "- %s\n", conn.Name()))
	}
	u.SkipV(fmt.Fprintf(&msg, "\n⚠️ All %d previous connections will be overwritten ⚠️\nProceed?\n", len(pl.Deck.Get())))

	r.bus.Publish(evt.NewFollowup(s3box.UserValidationAsked{
		Reason:  nextEvt,
		Message: msg.String(),
	}))
}

func (r *FyneConnectionsRepository) handleUserValidationAccepted(evt event.Event) {
	pl := evt.Payload().(s3box.UserValidationAccepted)
	reasonPl, ok := pl.Reason.Payload().(connection_deck.ImportConfirmationAsked)
	if !ok {
		return
	}

	dtos := dto.NewConnectionsDTOFromList(reasonPl.NewConnections)
	jsonContent, err := json.Marshal(dtos)
	if err != nil {
		r.bus.Publish(pl.Reason.NewFollowup(connection_deck.ImportFailed{
			Deck: reasonPl.Deck,
			Err:  err,
		}))
		return
	}
	r.prefs.SetString(allConnectionsKey, string(jsonContent))
	r.bus.Publish(pl.Reason.NewFollowup(connection_deck.ImportSucceeded(reasonPl)))
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
	pl := evt.Payload().(connection_deck.SelectConnectionTriggered)
	if err := r.saveDeck(ctx, pl.Deck); err != nil {
		r.bus.Publish(evt.NewFollowup(connection_deck.SelectConnectionFailed{
			Err:               err,
			ConnectionPayload: pl.ConnectionPayload,
		}))
	}
	r.bus.Publish(evt.NewFollowup(connection_deck.SelectConnectionSucceeded{
		ConnectionPayload: pl.ConnectionPayload,
		Deck:              pl.Deck,
	}))
}

func (r *FyneConnectionsRepository) handleCreate(evt event.Event) {
	ctx := evt.Context()
	pl := evt.Payload().(connection_deck.CreateConnectionTriggered)
	if err := r.saveDeck(ctx, pl.Deck); err != nil {
		r.bus.Publish(evt.NewFollowup(connection_deck.CreateConnectionFailed{
			Err:               err,
			ConnectionPayload: pl.ConnectionPayload,
		}))
	}
	r.bus.Publish(evt.NewFollowup(connection_deck.CreateConnectionSucceeded(pl)))
}

func (r *FyneConnectionsRepository) handleRemove(evt event.Event) {
	ctx := evt.Context()
	pl := evt.Payload().(connection_deck.RemoveConnectionTriggered)
	if err := r.saveDeck(ctx, pl.Deck); err != nil {
		r.bus.Publish(evt.NewFollowup(connection_deck.RemoveConnectionFailed{
			ConnectionPayload: pl.ConnectionPayload,
			Err:               err,
			RemovedIndex:      pl.RemovedIndex,
			WasSelected:       pl.WasSelected,
		}))
	}
	r.bus.Publish(evt.NewFollowup(connection_deck.RemoveConnectionSucceeded{
		ConnectionPayload: pl.ConnectionPayload,
		Deck:              pl.Deck,
	}))
}

func (r *FyneConnectionsRepository) handleUpdate(evt event.Event) {
	ctx := evt.Context()
	pl := evt.Payload().(connection_deck.UpdateConnectionTriggered)
	if err := r.saveDeck(ctx, pl.Deck); err != nil {
		r.bus.Publish(evt.NewFollowup(connection_deck.UpdateConnectionFailed{
			ConnectionPayload: pl.ConnectionPayload,
			Err:               err,
		}))
	}
	r.bus.Publish(evt.NewFollowup(connection_deck.UpdateConnectionSucceeded{
		ConnectionPayload: pl.ConnectionPayload,
		Deck:              pl.Deck,
	}))
}
