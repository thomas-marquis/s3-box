package viewmodel

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

var connNotInBindingErr = fmt.Errorf("connection not found in binding list")

type ConnectionViewModel interface {
	// Connections returns the list of connections as a binding.UntypedList
	Connections() binding.UntypedList

	// Save saves a connection to the repository and updates the binding list
	Save(c connection.Connection) error

	// Delete deletes a connection from the repository and updates the binding list
	Delete(c *connection.Connection) error

	// Select selects a connection and returns true if a new connection was successfully selected
	// and false if the set connection is the same as the current connection
	Select(c *connection.Connection) (bool, error)

	// ExportAsJSON exports all connections as a JSON object (byte slice)
	ExportAsJSON() (connection.ConnectionExport, error)

	// IsLoading returns true if the current selected connection is in read only mode
	IsReadOnly() bool
}

type connectionViewModelImpl struct {
	connRepo           connection.Repository
	settingsVm         SettingsViewModel
	connections        binding.UntypedList
	selectedConnection *connection.Connection
	loading            binding.Bool
}

var _ ConnectionViewModel = &connectionViewModelImpl{}

func NewConnectionViewModel(
	connRepo connection.Repository,
	settingsVm SettingsViewModel,
) *connectionViewModelImpl {
	c := binding.NewUntypedList()

	vm := &connectionViewModelImpl{
		connRepo:    connRepo,
		settingsVm:  settingsVm,
		connections: c,
		loading:     binding.NewBool(),
	}

	if err := vm.loadInitialConnections(); err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error refreshing connections: %w", err)
		fmt.Printf("error refreshing connections: %v", err)
	}

	vm.loading.Set(false)

	return vm
}

func (c *connectionViewModelImpl) Connections() binding.UntypedList {
	return c.connections
}

func (vm *connectionViewModelImpl) Save(c connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Save(ctx, &c); err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error saving connection: %w", err)
		fmt.Printf("error saving connection: %v", err)
		return err
	}

	if err := vm.updateBinding(&c); err != nil {
		if err == connNotInBindingErr {
			vm.connections.Append(&c)
		} else {
			return err
		}
	}

	return nil
}

func (vm *connectionViewModelImpl) Delete(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Delete(ctx, c.ID); err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error deleting connection: %w", err)
		fmt.Printf("error deleting connection: %v", err)
		return err
	}

	prevConns, err := uiutils.GetUntypedList[*connection.Connection](vm.connections)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error getting previous connections: %w", err)
		fmt.Printf("error getting previous connections: %v", err)
		return err
	}

	found := false
	for _, prevConn := range prevConns {
		if prevConn.ID == c.ID {
			found = vm.connections.Remove(prevConn) == nil
		}
	}

	if !found {
		// Is this case possible? If so, we should handle it gracefully.
		return fmt.Errorf("connection with ID %s not found", c.ID)
	}

	return nil
}

func (vm *connectionViewModelImpl) Select(c *connection.Connection) (bool, error) {
	vm.loading.Set(true)
	defer vm.loading.Set(false)
	prevSelectedConn := vm.selectedConnection

	if prevSelectedConn != nil && prevSelectedConn.ID == c.ID {
		fmt.Println("Connection is already selected, no change made.") // TODO: remove
		return false, nil
	}

	fmt.Println("COUCOU") // TODO: remove

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.SetSelected(ctx, c.ID); err != nil {
		return false, err
	}
	prevSelectedConn.IsSelected = false
	c.IsSelected = true
	vm.selectedConnection = c

	if err := vm.updateBinding(c); err != nil {
		prevSelectedConn.IsSelected = true
		c.IsSelected = false
		vm.selectedConnection = prevSelectedConn
		return false, err
	}

	return true, nil
}

func (vm *connectionViewModelImpl) ExportAsJSON() (connection.ConnectionExport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	return vm.connRepo.ExportToJson(ctx)
}

func (vm *connectionViewModelImpl) IsReadOnly() bool {
	if vm.selectedConnection == nil {
		return false
	}
	return vm.selectedConnection.ReadOnly
}

func (vm *connectionViewModelImpl) updateBinding(c *connection.Connection) error {
	allConns, err := uiutils.GetUntypedList[*connection.Connection](vm.connections)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error getting previous connections: %w", err)
		fmt.Printf("error listing connections: %v", err)
		return err
	}

	found := false
	for i, conn := range allConns {
		if conn.ID == c.ID {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := vm.connections.SetValue(i, &updatedConn); err != nil {
				// TOOD: send to global logging chan
				// vm.errChan <- fmt.Errorf("error setting selected connection: %w", err)
				fmt.Printf("error updating connection: %v", err)
				return err
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholcerConn := connection.NewEmptyConnection()
			vm.connections.Append(placeholcerConn)
			vm.connections.Remove(placeholcerConn)
		}
	}

	if !found {
		return connNotInBindingErr
	}

	return nil
}

func (vm *connectionViewModelImpl) loadInitialConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	conns, err := vm.connRepo.List(ctx)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error listing connections: %w", err)
		fmt.Printf("error listing connections: %v", err)
		return err
	}

	for _, c := range conns {
		vm.connections.Append(c)
		if c.IsSelected {
			vm.selectedConnection = c
		}
	}

	return nil
}
