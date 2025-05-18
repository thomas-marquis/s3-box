package viewmodel

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/connection"
)

type ConnectionViewModel struct {
	connRepo           connection.Repository
	connSvc            connection.ConnectionService
	settingsVm         SettingsViewModel
	connections        binding.UntypedList
	selectedConnection *connection.Connection
	loading            binding.Bool
}

func NewConnectionViewModel(connRepo connection.Repository, connSvc connection.ConnectionService, settingsVm SettingsViewModel) *ConnectionViewModel {
	c := binding.NewUntypedList()

	vm := &ConnectionViewModel{
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

func (c *ConnectionViewModel) Connections() binding.UntypedList {
	return c.connections
}

func (vm *ConnectionViewModel) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	conns, err := vm.connRepo.ListConnections(ctx)
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
	}

	return nil
}

func (vm *ConnectionViewModel) SaveConnection(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.SaveConnection(ctx, c); err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error saving connection: %w", err)
		fmt.Printf("error saving connection: %v", err)
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ConnectionViewModel) DeleteConnection(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.DeleteConnection(ctx, c.ID); err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error deleting connection: %w", err)
		fmt.Printf("error deleting connection: %v", err)
		return err
	}

	return vm.RefreshConnections()
}

// SelectConnection selects a connection and returns true if a new connection was successfully selected
// and false if the set connection is the same as the current connection
func (vm *ConnectionViewModel) SelectConnection(c *connection.Connection) (bool, error) {
	vm.loading.Set(true)
	defer vm.loading.Set(false)
	prevConn := vm.selectedConnection

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.SetSelectedConnection(ctx, c.ID); err != nil {
		return false, err
	}

	vm.selectedConnection = c
	return c != prevConn, nil
}

func (vm *ConnectionViewModel) ExportConnectionsAsJSON() (connection.ConnectionExport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	return vm.connRepo.ExportToJson(ctx)
}
