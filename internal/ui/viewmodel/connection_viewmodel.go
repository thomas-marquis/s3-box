package viewmodel

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
	"io"
)

var errConnNotInBinding = fmt.Errorf("connection not found in binding list")

// ConnectionViewModel provides methods to manage, update, and query connections within the application.
type ConnectionViewModel interface {
	////////////////////////
	// State methods
	////////////////////////

	// Connections return the list of connections as a binding.UntypedList
	Connections() binding.UntypedList

	// Deck return user's connections deck
	Deck() *connection_deck.Deck

	// ErrorMessages returns a binding.String that provides access to the current error messages for the connection view model.
	// When an error occurred, the contained string is set.
	ErrorMessages() binding.String

	// IsReadOnly returns true if the connection view model is in a read-only state, otherwise false.
	IsReadOnly() bool

	Loading() binding.Bool

	IsLoading() bool

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
	connectionRepository connection_deck.Repository
	settingsViewModel    SettingsViewModel
	connBindings         binding.UntypedList
	deck                 *connection_deck.Deck
	notifier             notification.Repository
	onChangeCallbacks    []func(*connection_deck.Connection)
	bus                  event.Bus
	errorMsgBinding      binding.String
	loading              binding.Bool
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
	errorMsgBinding.Set("")

	loading := binding.NewBool()
	loading.Set(false)

	vm := &connectionViewModelImpl{
		connectionRepository: connectionRepository,
		settingsViewModel:    settingsViewModel,
		connBindings:         c,
		deck:                 deck,
		notifier:             notifier,
		onChangeCallbacks:    make([]func(*connection_deck.Connection), 0),
		bus:                  bus,
		errorMsgBinding:      errorMsgBinding,
		loading:              loading,
	}

	vm.initConnections(deck)

	bus.Publish(connection_deck.NewSelectEvent(deck, deck.SelectedConnection(), nil))

	go vm.listenUiEvents()

	return vm
}

func (vm *connectionViewModelImpl) Connections() binding.UntypedList {
	return vm.connBindings
}

func (vm *connectionViewModelImpl) Deck() *connection_deck.Deck {
	return vm.deck
}

func (vm *connectionViewModelImpl) ErrorMessages() binding.String {
	return vm.errorMsgBinding
}

func (vm *connectionViewModelImpl) Loading() binding.Bool {
	return vm.loading
}

func (vm *connectionViewModelImpl) IsLoading() bool {
	val, _ := vm.loading.Get()
	return val
}

func (vm *connectionViewModelImpl) Update(
	connID connection_deck.ConnectionID,
	options ...connection_deck.ConnectionOption,
) {
	evt, err := vm.deck.Update(connID, options...)
	if err != nil {
		vm.bus.Publish(connection_deck.NewUpdateFailureEvent(
			fmt.Errorf("impossible to update connection %s in user's deck: %w", connID, err),
			vm.findConnectionInBinding(connID)))
		return
	}
	vm.bus.Publish(evt)
}

func (vm *connectionViewModelImpl) Delete(conn *connection_deck.Connection) {
	evt, err := vm.deck.RemoveAConnection(conn.ID())
	if err != nil {
		vm.bus.Publish(connection_deck.NewRemoveFailureEvent(err, 0, false, conn))
	}
	vm.bus.Publish(evt)
}

func (vm *connectionViewModelImpl) Create(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) {
	evt := vm.deck.New(name, accessKey, secretKey, bucket, options...)
	vm.bus.Publish(evt)
}

func (vm *connectionViewModelImpl) Select(conn *connection_deck.Connection) {
	evt, err := vm.deck.Select(conn.ID())
	if err != nil {
		vm.bus.Publish(connection_deck.NewSelectFailureEvent(err, conn))
	}
	vm.bus.Publish(evt)
}

func (vm *connectionViewModelImpl) ExportAsJSON(writer io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsViewModel.CurrentTimeout())
	defer cancel()
	if err := vm.connectionRepository.Export(ctx, writer); err != nil {
		return vm.notifier.NotifyError(err)
	}

	return nil
}

func (vm *connectionViewModelImpl) IsReadOnly() bool {
	if vm.deck.SelectedConnection() == nil {
		return false
	}
	return vm.deck.SelectedConnection().ReadOnly()
}

func (vm *connectionViewModelImpl) deleteFromBinding(deletedConn *connection_deck.Connection) {
	found := false
	currentConns := vm.deck.Get()
	for _, prevConn := range currentConns {
		if prevConn.Is(deletedConn) {
			found = vm.connBindings.Remove(prevConn) == nil
		}
	}

	if !found {
		vm.bus.Publish(connection_deck.NewRemoveFailureEvent(
			errConnNotInBinding, len(currentConns), false, deletedConn))
	}
}

func (vm *connectionViewModelImpl) findConnectionInBinding(connID connection_deck.ConnectionID) *connection_deck.Connection {
	connections, err := uiutils.GetUntypedList[*connection_deck.Connection](vm.connBindings)
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

func (vm *connectionViewModelImpl) updateConnectionBinding(c *connection_deck.Connection) {
	found := false
	for i, conn := range vm.deck.Get() {
		if conn.Is(c) {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := vm.connBindings.SetValue(i, &updatedConn); err != nil {
				vm.bus.Publish(connection_deck.NewUpdateFailureEvent(err, vm.findConnectionInBinding(c.ID())))
				return
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholderConn := connection_deck.Connection{}
			vm.connBindings.Append(placeholderConn)
			vm.connBindings.Remove(placeholderConn)
		}
	}

	if !found {
		vm.bus.Publish(connection_deck.NewUpdateFailureEvent(errConnNotInBinding, nil))
	}
}

func (vm *connectionViewModelImpl) initConnections(deck *connection_deck.Deck) {
	for _, c := range deck.Get() {
		vm.connBindings.Append(c)
	}
}

func (vm *connectionViewModelImpl) listenUiEvents() {
	for evt := range vm.bus.Subscribe() {
		switch evt.Type() {
		case connection_deck.SelectEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case connection_deck.SelectEventType.AsFailure():
			e := evt.(connection_deck.SelectFailureEvent)
			vm.errorMsgBinding.Set(e.Error().Error())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.SelectEventType.AsSuccess():
			e := evt.(connection_deck.SelectSuccessEvent)
			vm.updateConnectionBinding(e.Connection())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.CreateEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case connection_deck.CreateEventType.AsFailure():
			e := evt.(connection_deck.CreateFailureEvent)
			vm.errorMsgBinding.Set(e.Error().Error())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.CreateEventType.AsSuccess():
			e := evt.(connection_deck.CreateSuccessEvent)
			vm.connBindings.Append(e.Connection())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.RemoveEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case connection_deck.RemoveEventType.AsFailure():
			e := evt.(connection_deck.RemoveFailureEvent)
			vm.errorMsgBinding.Set(e.Error().Error())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.RemoveEventType.AsSuccess():
			e := evt.(connection_deck.RemoveSuccessEvent)
			vm.deleteFromBinding(e.Connection())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.UpdateEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case connection_deck.UpdateEventType.AsFailure():
			e := evt.(connection_deck.UpdateFailureEvent)
			vm.errorMsgBinding.Set(e.Error().Error())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		case connection_deck.UpdateEventType.AsSuccess():
			e := evt.(connection_deck.UpdateSuccessEvent)
			vm.updateConnectionBinding(e.Connection())
			vm.deck.Notify(evt)
			vm.loading.Set(false)

		}
	}
}
