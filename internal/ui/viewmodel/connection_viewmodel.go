package viewmodel

import (
	"context"
	"fmt"
	"io"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

var errConnNotInBinding = fmt.Errorf("connection not found in binding list")

// ConnectionViewModel provides methods to manage, update, and query connections within the application.
type ConnectionViewModel interface {
	ViewModel

	////////////////////////
	// State methods
	////////////////////////

	// Connections return the list of connections as a binding.UntypedList
	Connections() binding.UntypedList

	// Deck return user's connections deck
	Deck() *connection_deck.Deck

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
	connBindings         binding.UntypedList
	deck                 *connection_deck.Deck
	notifier             notification.Repository
	onChangeCallbacks    []func(*connection_deck.Connection)
	bus                  event.Bus
}

func NewConnectionViewModel(
	connectionRepository connection_deck.Repository,
	settingsViewModel SettingsViewModel,
	notifier notification.Repository,
	bus event.Bus,
) ConnectionViewModel {
	c := binding.NewUntypedList()

	ctx, cancel := context.WithTimeout(context.Background(), settingsViewModel.CurrentTimeout())
	defer cancel()

	deck, err := connectionRepository.Get(ctx)
	if err != nil {
		notifier.NotifyError(fmt.Errorf("error getting initial connections: %w", err))
		return nil
	}

	errorMsgBinding := binding.NewString()
	errorMsgBinding.Set("") //nolint:errcheck

	loading := binding.NewBool()
	loading.Set(false) //nolint:errcheck

	vm := &connectionViewModelImpl{
		baseViewModel: baseViewModel{
			loading:      binding.NewBool(),
			errorMessage: binding.NewString(),
			infoMessage:  binding.NewString(),
		},
		connectionRepository: connectionRepository,
		settingsViewModel:    settingsViewModel,
		connBindings:         c,
		deck:                 deck,
		notifier:             notifier,
		onChangeCallbacks:    make([]func(*connection_deck.Connection), 0),
		bus:                  bus,
	}

	vm.initConnections(deck)

	//bus.Publish(connection_deck.NewSelectEvent(deck, deck.SelectedConnection(), nil))
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

func (v *connectionViewModelImpl) Connections() binding.UntypedList {
	return v.connBindings
}

func (v *connectionViewModelImpl) Deck() *connection_deck.Deck {
	return v.deck
}

func (v *connectionViewModelImpl) Update(
	connID connection_deck.ConnectionID,
	options ...connection_deck.ConnectionOption,
) {
	evt, err := v.deck.Update(connID, options...)
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) Select(conn *connection_deck.Connection) {
	evt, err := v.deck.Select(conn.ID())
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) handleUpdate(evt event.Event) {
	cg := evt.Payload.(connection_deck.ConnectionGetter)
	v.updateConnectionBinding(evt, cg.Connection())
	v.deck.Notify(evt)
	v.loading.Set(false) //nolint:errcheck
}

func (v *connectionViewModelImpl) Delete(conn *connection_deck.Connection) {
	evt, err := v.deck.RemoveAConnection(conn.ID())
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) handleDelete(evt event.Event) {
	pl := evt.Payload.(connection_deck.RemoveConnectionSucceeded)
	if err := v.deleteFromBinding(evt, pl.Connection()); err != nil {
		return
	}
	v.deck.Notify(evt)
	v.loading.Set(false) //nolint:errcheck
}

func (v *connectionViewModelImpl) Create(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) {
	evt := v.deck.New(name, accessKey, secretKey, bucket, options...)
	v.bus.Publish(evt)
}

func (v *connectionViewModelImpl) handleCreate(evt event.Event) {
	pl := evt.Payload.(connection_deck.CreateConnectionSucceeded)
	v.connBindings.Append(pl.Connection()) //nolint:errcheck
	v.deck.Notify(evt)
	v.loading.Set(false) //nolint:errcheck
}

func (v *connectionViewModelImpl) ExportAsJSON(writer io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), v.settingsViewModel.CurrentTimeout())
	defer cancel()
	if err := v.connectionRepository.Export(ctx, writer); err != nil {
		v.notifier.NotifyError(err)
		return err
	}

	return nil
}

func (v *connectionViewModelImpl) IsReadOnly() bool {
	if v.deck.SelectedConnection() == nil {
		return false
	}
	return v.deck.SelectedConnection().ReadOnly()
}

func (v *connectionViewModelImpl) deleteFromBinding(evt event.Event, deletedConn *connection_deck.Connection) error {
	found := false
	allConnections := uiutils.GetUntypedListOrPanic[*connection_deck.Connection](v.connBindings)
	for _, prevConn := range allConnections {
		if prevConn.Is(deletedConn) {
			found = v.connBindings.Remove(prevConn) == nil
		}
	}

	if !found {
		v.bus.Publish(event.NewFollowup(evt, connection_deck.RemoveConnectionFailed{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: deletedConn},
			Err:               errConnNotInBinding,
			RemovedIndex:      len(allConnections),
			WasSelected:       false,
		}))
		return errConnNotInBinding
	}

	return nil
}

func (v *connectionViewModelImpl) findConnectionInBinding(connID connection_deck.ConnectionID) *connection_deck.Connection {
	connections, err := uiutils.GetUntypedList[*connection_deck.Connection](v.connBindings)
	if err != nil {
		return nil
	}

	for _, conn := range connections {
		if conn.ID() == connID {
			return conn
		}
	}
	return nil
}

func (v *connectionViewModelImpl) updateConnectionBinding(evt event.Event, c *connection_deck.Connection) {
	found := false
	for i, conn := range v.deck.Get() {
		if conn.Is(c) {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := v.connBindings.SetValue(i, &updatedConn); err != nil {
				v.bus.Publish(event.NewFollowup(evt, connection_deck.UpdateConnectionFailed{
					ConnectionPayload: connection_deck.ConnectionPayload{Conn: v.findConnectionInBinding(c.ID())},
					Err:               err,
				}))
				return
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholderConn := connection_deck.Connection{}
			v.connBindings.Append(placeholderConn) //nolint:errcheck
			v.connBindings.Remove(placeholderConn) //nolint:errcheck
		}
	}

	if !found {
		v.bus.Publish(event.NewFollowup(evt, connection_deck.UpdateConnectionFailed{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: nil},
			Err:               errConnNotInBinding,
		}))
		return
	}
}

func (v *connectionViewModelImpl) initConnections(deck *connection_deck.Deck) {
	for _, c := range deck.Get() {
		v.connBindings.Append(c) //nolint:errcheck
	}
}

func (v *connectionViewModelImpl) handleOnLoading(_ event.Event) {
	if v.IsLoading() {
		return
	}
	v.loading.Set(true) //nolint:errcheck
}

func (v *connectionViewModelImpl) handleFailure(evt event.Event) {
	pl := evt.Payload.(connection_deck.ErrorGetter)
	v.errorMessage.Set(pl.Error().Error()) //nolint:errcheck
	v.deck.Notify(evt)
	v.loading.Set(false) //nolint:errcheck
}
