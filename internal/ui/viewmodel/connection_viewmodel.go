package viewmodel

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type ConnectionViewModel interface {
	Connections() binding.UntypedList
	RefreshConnections() error
	SaveConnection(c connection.Connection) error
	DeleteConnection(c *connection.Connection) error

	// SelectConnection selects a connection and returns true if a new connection was successfully selected
	// and false if the set connection is the same as the current connection
	SelectConnection(c *connection.Connection) (bool, error)

	ExportConnectionsAsJSON() (connection.ConnectionExport, error)

	// IsLoading returns true if the current selected connection is in read only mode
	IsReadOnly() bool
}

type connectionViewModelImpl struct {
	connRepo           connection.Repository
	connSvc            connection.ConnectionService
	settingsVm         SettingsViewModel
	connections        binding.UntypedList
	selectedConnection *connection.Connection
	loading            binding.Bool
}

var _ ConnectionViewModel = &connectionViewModelImpl{}

func NewConnectionViewModel(
	connRepo connection.Repository,
	connSvc connection.ConnectionService,
	settingsVm SettingsViewModel,
) *connectionViewModelImpl {
	c := binding.NewUntypedList()

	vm := &connectionViewModelImpl{
		connRepo:    connRepo,
		connSvc:     connSvc,
		settingsVm:  settingsVm,
		connections: c,
		loading:     binding.NewBool(),
	}

	if err := vm.RefreshConnections(); err != nil {
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

func (vm *connectionViewModelImpl) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	conns, err := vm.connRepo.List(ctx)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error listing connections: %w", err)
		fmt.Printf("error listing connections: %v", err)
		return err
	}

	prevConns, err := vm.connections.Get()
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error getting previous connections: %w", err)
		fmt.Printf("error getting previous connections: %v", err)
		return err
	}
	for _, c := range prevConns {
		vm.connections.Remove(c)
	}

	for _, c := range conns {
		vm.connections.Append(c)
		if c.IsSelected {
			vm.selectedConnection = c
		}
	}

	return nil
}

func (vm *connectionViewModelImpl) SaveConnection(c connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Save(ctx, &c); err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error saving connection: %w", err)
		fmt.Printf("error saving connection: %v", err)
		return err
	}

	existingConns, err := uiutils.GetUntypedList[*connection.Connection](vm.connections)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error getting existing connections: %w", err)
		fmt.Printf("error getting existing connections: %v", err)
		return err
	}

	for _, existingConn := range existingConns {
		if existingConn.ID == c.ID {
			fmt.Printf("Update connection (view data): %v\n", existingConn) // TODO remove it
			vm.connections.Remove(existingConn)
			vm.connections.Append(&c)
		}
	}

	return nil
}

func (vm *connectionViewModelImpl) DeleteConnection(c *connection.Connection) error {
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

func (vm *connectionViewModelImpl) SelectConnection(c *connection.Connection) (bool, error) {
	vm.loading.Set(true)
	defer vm.loading.Set(false)
	prevSelectedConn := vm.selectedConnection

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.SetSelected(ctx, c.ID); err != nil {
		return false, err
	}

	allConns, err := uiutils.GetUntypedList[*connection.Connection](vm.connections)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error getting previous connections: %w", err)
		fmt.Printf("error getting previous connections: %v", err)
		return false, err
	}

	found := false
	for i, conn := range allConns {
		if conn.ID == c.ID {
			found = true
			selectedConn := *conn // Create a copy to have a new ref in the binding
			selectedConn.IsSelected = true
			prevSelectedConn.IsSelected = false
			fmt.Printf("old conn pointer (before): %p | new conn pointer: %p\n", prevSelectedConn, &selectedConn) // TODO remove it
			vm.connections.SetValue(i, &selectedConn)
			vm.selectedConnection = &selectedConn
		}
		if conn.ID == prevSelectedConn.ID {
			conn.IsSelected = false
		}
	}

	if !found {
		// Is this case possible? If so, we should handle it gracefully.
		return false, fmt.Errorf("connection with ID %s not found", c.ID)
	}

	return c.ID != prevSelectedConn.ID, nil
}

func (vm *connectionViewModelImpl) ExportConnectionsAsJSON() (connection.ConnectionExport, error) {
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
