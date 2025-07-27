package viewmodel

import (
	"context"
	"fmt"
	"io"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

var errConnNotInBinding = fmt.Errorf("connection not found in binding list")

// ConnectionViewModel provides methods to manage, update, and query connections within the application.
type ConnectionViewModel interface {
	// Connections return the list of connections as a binding.UntypedList
	Connections() binding.UntypedList

	// Deck return user's connections deck
	Deck() *connection_deck.Deck

	// Create adds a new connection with the specified name, access key, secret key, bucket, and optional settings.
	// Returns an error if the creation process fails.
	Create(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) error

	// Update updates the connection with the specified connection ID using the provided options. Returns an error on failure.
	Update(connID connection_deck.ConnectionID, options ...connection_deck.ConnectionOption) error

	// Delete deletes the specified connection
	Delete(connID connection_deck.ConnectionID) error

	// Select selects a connection and returns true if a new connection was successfully selected
	// and false if the set connection is the same as the current connection
	Select(c *connection_deck.Connection) (bool, error)

	// ExportAsJSON exports all connections JSON serialized.
	// The JSON object will be written in the writer.
	// It's up to you to effectively write the writer into a file or whatever.
	ExportAsJSON(writer io.Writer) error

	// IsReadOnly returns true if the connection view model is in a read-only state, otherwise false.
	IsReadOnly() bool
}

type connectionViewModelImpl struct {
	connRepo     connection_deck.Repository
	settingsVm   SettingsViewModel
	connBindings binding.UntypedList
	deck         *connection_deck.Deck
	errorStream  chan<- error
}

func NewConnectionViewModel(
	connRepo connection_deck.Repository,
	settingsVm SettingsViewModel,
	errorStream chan<- error,
) ConnectionViewModel {
	c := binding.NewUntypedList()

	ctx, cancel := context.WithTimeout(context.Background(), settingsVm.CurrentTimeout())
	defer cancel()

	deck, err := connRepo.Get(ctx)
	if err != nil {
		panic(fmt.Errorf("error getting initial connections: %w", err))
	}

	vm := &connectionViewModelImpl{
		connRepo:     connRepo,
		settingsVm:   settingsVm,
		connBindings: c,
		deck:         deck,
		errorStream:  errorStream,
	}

	if err := vm.initConnections(deck); err != nil {
		// TOOD: send to global logging chan
		// vm.errorStream <- fmt.Errorf("error refreshing connections: %w", err)
		fmt.Printf("error refreshing connections: %v", err)
	}

	return vm
}

func (vm *connectionViewModelImpl) Connections() binding.UntypedList {
	return vm.connBindings
}

func (vm *connectionViewModelImpl) Deck() *connection_deck.Deck {
	return vm.deck
}

func (vm *connectionViewModelImpl) Create(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) error {
	vm.deck.New(name, accessKey, secretKey, bucket, options...)
	if err := vm.sync(); err != nil {
		vm.errorStream <- err
		return err
	}
	return nil
}

func (vm *connectionViewModelImpl) Update(connID connection_deck.ConnectionID, options ...connection_deck.ConnectionOption) error {
	conn, err := vm.deck.GetByID(connID)
	if err != nil {
		err = fmt.Errorf("connection %s not found in user's deck: %w", connID, err)
		vm.errorStream <- err
		return err
	}

	for _, option := range options {
		option(conn)
	}

	if err := vm.sync(); err != nil {
		vm.errorStream <- err
		return err
	}
	return nil
}

func (vm *connectionViewModelImpl) Delete(connID connection_deck.ConnectionID) error {
	if err := vm.deck.RemoveAConnection(connID); err != nil {
		err = fmt.Errorf("error deleting connection from set: %w", err)
		vm.errorStream <- err
		return err
	}

	if err := vm.sync(); err != nil {
		vm.errorStream <- err
		return err
	}

	prevConns, err := uiutils.GetUntypedList[*connection_deck.Connection](vm.connBindings)
	if err != nil {
		vm.errorStream <- err
		return err
	}

	found := false
	for _, prevConn := range prevConns {
		if prevConn.ID() == connID {
			found = vm.connBindings.Remove(prevConn) == nil
		}
	}

	if !found {
		err = fmt.Errorf("connection %s not found in user's deck: %w", connID, err)
		vm.errorStream <- err
		return err
	}

	return nil
}

func (vm *connectionViewModelImpl) Select(c *connection_deck.Connection) (bool, error) {
	prevSelected := vm.deck.SelectedConnection()

	if err := vm.deck.Select(c.ID()); err != nil {
		err = fmt.Errorf("error selecting connection: %w", err)
		vm.errorStream <- err
		return false, err
	}

	if err := vm.sync(); err != nil {
		if prevSelected != nil {
			vm.deck.Select(prevSelected.ID())
		}
		err = fmt.Errorf("error saving connection set: %w", err)
		vm.errorStream <- err
		return false, err
	}

	if err := vm.updateBinding(c); err != nil {
		if prevSelected != nil {
			newSelected := vm.deck.SelectedConnection()
			vm.deck.Select(prevSelected.ID())
			if err := vm.sync(); err != nil {
				vm.deck.Select(newSelected.ID())
				err = fmt.Errorf("error saving connection set after selection: %w", err)
				vm.errorStream <- err
				return false, err
			}
		}
		vm.errorStream <- err
		return false, err
	}

	return true, nil
}

func (vm *connectionViewModelImpl) ExportAsJSON(writer io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Export(ctx, writer); err != nil {
		vm.errorStream <- err
		return err
	}

	return nil
}

func (vm *connectionViewModelImpl) IsReadOnly() bool {
	if vm.deck.SelectedConnection() == nil {
		return false
	}
	return vm.deck.SelectedConnection().ReadOnly()
}

func (vm *connectionViewModelImpl) updateBinding(c *connection_deck.Connection) error {
	found := false
	for i, conn := range vm.deck.Get() {
		if conn.Is(c) {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := vm.connBindings.SetValue(i, &updatedConn); err != nil {
				vm.errorStream <- err
				return err
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholderConn := connection_deck.Connection{}
			vm.connBindings.Append(placeholderConn)
			vm.connBindings.Remove(placeholderConn)
		}
	}

	if !found {
		vm.errorStream <- errConnNotInBinding
		return errConnNotInBinding
	}

	return nil
}

func (vm *connectionViewModelImpl) initConnections(deck *connection_deck.Deck) error {
	for _, c := range deck.Get() {
		vm.connBindings.Append(c)
	}

	return nil
}

// sync saves the current deck state to the repository.
// Returns an error if the save operation fails.
func (vm *connectionViewModelImpl) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Save(ctx, vm.deck); err != nil {
		return err
	}
	return nil
}
