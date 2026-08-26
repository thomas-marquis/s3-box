package viewmodel

import (
	"context"
	"fmt"
	"io"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
)

var errConnNotInBinding = fmt.Errorf("connection not found in binding list")

// ConnectionViewModel provides methods to manage, update, and query connections within the application.
type ConnectionViewModel interface {
	ViewModel

	////////////////////////
	// State methods
	////////////////////////

	// IsReadOnly returns true if the connection view model is in a read-only state, otherwise false.
	IsReadOnly() bool

	////////////////////////
	// Action methods
	////////////////////////

	Select(conn *connection_deck.Connection)

	Delete(conn *connection_deck.Connection)

	Create(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption)

	// Update updates the connection with the specified connection ID using the provided options. Returns an error on failure.
	Update(connID connection_deck.ConnectionID, options ...connection_deck.ConnectionOption)

	// ExportAsJSON exports all connections JSON serialized.
	// The JSON object will be written in the writer.
	// It's up to you to effectively write the writer into a file or whatever.
	ExportAsJSON(writer io.Writer) error
}

type connectionViewModelImpl struct {
	baseViewModel

	connectionRepository connection_deck.Repository
	settingsViewModel    SettingsViewModel
	state                *state.State
	notifier             notification.Repository
	onChangeCallbacks    []func(*connection_deck.Connection)
	bus                  event.Bus
}

func NewConnectionViewModel(
	connectionRepository connection_deck.Repository,
	settingsViewModel SettingsViewModel,
	appState *state.State,
	notifier notification.Repository,
	bus event.Bus,
) ConnectionViewModel {
	ctx, cancel := context.WithTimeout(context.Background(), appState.Settings().TimeoutValue())
	defer cancel()

	deck, err := connectionRepository.Get(ctx)
	if err != nil {
		notifier.NotifyError(fmt.Errorf("error getting initial connections: %w", err))
		return nil
	}

	errorMsgBinding := binding.NewString()
	u.Skip(errorMsgBinding.Set(""))

	loading := binding.NewBool()
	u.Skip(loading.Set(false))

	vm := &connectionViewModelImpl{
		baseViewModel: baseViewModel{
			loading:      binding.NewBool(),
			errorMessage: binding.NewString(),
			infoMessage:  binding.NewString(),
		},
		connectionRepository: connectionRepository,
		settingsViewModel:    settingsViewModel,
		state:                appState,
		notifier:             notifier,
		onChangeCallbacks:    make([]func(*connection_deck.Connection), 0),
		bus:                  bus,
	}

	appState.Connection().Init(deck)

	bus.Publish(event.New(connection_deck.SelectConnectionTriggered{
		ConnectionPayload: connection_deck.ConnectionPayload{Conn: deck.SelectedConnection()},
		Deck:              deck,
	}))

	vm.bus.Subscribe().
		On(event.IsOneOf(
			connection_deck.SelectConnectionTriggeredType,
			connection_deck.CreateConnectionTriggeredType,
			connection_deck.RemoveConnectionTriggeredType,
			connection_deck.UpdateConnectionTriggeredType,
		), vm.handleOnLoading).
		On(event.IsOneOf(
			connection_deck.SelectConnectionFailedType,
			connection_deck.CreateConnectionFailedType,
			connection_deck.RemoveConnectionFailedType,
			connection_deck.UpdateConnectionFailedType,
		), vm.handleFailure).
		On(event.IsOneOf(
			connection_deck.SelectConnectionSucceededType,
			connection_deck.UpdateConnectionSucceededType,
		), vm.handleUpdate).
		On(event.Is(connection_deck.CreateConnectionSucceededType), vm.handleCreate).
		On(event.Is(connection_deck.RemoveConnectionSucceededType), vm.handleDelete).
		ListenWithWorkers(1)

	return vm
}

func (v *connectionViewModelImpl) Update(
	connID connection_deck.ConnectionID,
	options ...connection_deck.ConnectionOption,
) {
	evt, err := v.state.Connection().Deck().Update(connID, options...)
	if err != nil {
		v.notifier.NotifyError(err)
		v.bus.Publish(event.New(connection_deck.UpdateConnectionFailed{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: v.state.Connection().FindOrNil(connID)},
			Err:               fmt.Errorf("impossible to update connection %s in user's deck: %w", connID, err),
		}))
		return
	}
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) Select(conn *connection_deck.Connection) {
	evt, err := v.state.Connection().Deck().Select(conn.ID())
	if err != nil {
		v.notifier.NotifyError(err)
		v.bus.Publish(event.New(connection_deck.SelectConnectionFailed{
			Err:               err,
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: conn},
		}))
		return
	}
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) handleUpdate(evt event.Event) {
	cg := evt.Payload().(connection_deck.ConnectionGetter)
	v.updateConnectionBinding(evt, cg.Connection())
	v.state.Connection().Deck().Notify(evt)
	u.Skip(v.loading.Set(false))
}

func (v *connectionViewModelImpl) Delete(conn *connection_deck.Connection) {
	evt, err := v.state.Connection().Deck().RemoveAConnection(conn.ID())
	if err != nil {
		v.notifier.NotifyError(err)
		v.bus.Publish(event.New(connection_deck.RemoveConnectionFailed{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: conn},
			Err:               err,
		}))
		return
	}
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) handleDelete(evt event.Event) {
	pl := evt.Payload().(connection_deck.RemoveConnectionSucceeded)
	if err := v.deleteFromBinding(evt, pl.Connection()); err != nil {
		return
	}
	v.state.Connection().Deck().Notify(evt)
	u.Skip(v.loading.Set(false))
}

func (v *connectionViewModelImpl) Create(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) {
	evt := v.state.Connection().Deck().New(name, accessKey, secretKey, bucket, options...)
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) handleCreate(evt event.Event) {
	pl := evt.Payload().(connection_deck.CreateConnectionSucceeded)
	u.Skip(v.state.Connection().List().Append(pl.Connection()))
	v.state.Connection().Deck().Notify(evt)
	u.Skip(v.loading.Set(false))
}

func (v *connectionViewModelImpl) ExportAsJSON(writer io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), v.state.Settings().TimeoutValue())
	defer cancel()
	if err := v.connectionRepository.Export(ctx, writer); err != nil {
		v.notifier.NotifyError(err)
		return err
	}

	return nil
}

func (v *connectionViewModelImpl) IsReadOnly() bool {
	if v.state.Connection().Deck().SelectedConnection() == nil {
		return false
	}
	return v.state.Connection().Deck().SelectedConnection().ReadOnly()
}

func (v *connectionViewModelImpl) deleteFromBinding(evt event.Event, deletedConn *connection_deck.Connection) error {
	found := false
	allConnections, _ := v.state.Connection().List().Get()
	for _, prevConn := range allConnections {
		if prevConn.Is(deletedConn) {
			found = v.state.Connection().List().Remove(prevConn) == nil
		}
	}

	if !found {
		v.bus.Publish(evt.NewFollowup(connection_deck.RemoveConnectionFailed{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: deletedConn},
			Err:               errConnNotInBinding,
			RemovedIndex:      len(allConnections),
			WasSelected:       false,
		}))
		return errConnNotInBinding
	}

	return nil
}

func (v *connectionViewModelImpl) updateConnectionBinding(evt event.Event, c *connection_deck.Connection) {
	found := false
	for i, conn := range v.state.Connection().Deck().Get() {
		if conn.Is(c) {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := v.state.Connection().List().SetValue(i, &updatedConn); err != nil {
				v.bus.Publish(evt.NewFollowup(connection_deck.UpdateConnectionFailed{
					ConnectionPayload: connection_deck.ConnectionPayload{Conn: v.state.Connection().FindOrNil(c.ID())},
					Err:               err,
				}))
				return
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholderConn := &connection_deck.Connection{}
			u.Skip(v.state.Connection().List().Append(placeholderConn))
			u.Skip(v.state.Connection().List().Remove(placeholderConn))
		}
	}

	if !found {
		v.bus.Publish(evt.NewFollowup(connection_deck.UpdateConnectionFailed{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: nil},
			Err:               errConnNotInBinding,
		}))
		return
	}
}

func (v *connectionViewModelImpl) handleOnLoading(_ event.Event) {
	if v.IsLoading() {
		return
	}
	u.Skip(v.loading.Set(true))
}

func (v *connectionViewModelImpl) handleFailure(evt event.Event) {
	pl := evt.Payload().(connection_deck.ErrorGetter)
	u.Skip(v.errorMessage.Set(pl.Error().Error()))
	v.state.Connection().Deck().Notify(evt)
	u.Skip(v.loading.Set(false))
}
