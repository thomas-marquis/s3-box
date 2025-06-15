package viewmodel

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/connections"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

var errConnNotInBinding = fmt.Errorf("connection not found in binding list")

type ConnectionViewModel interface {
	// Connections returns the list of connections as a binding.UntypedList
	Connections() binding.UntypedList

	Create(name, accessKey, secretKey, bucket string, options ...connections.ConnectionOption) error

	Update(c *connections.Connection, options ...connections.ConnectionOption) error

	// Delete deletes a connection from the repository and updates the binding list
	Delete(c *connections.Connection) error

	// Select selects a connection and returns true if a new connection was successfully selected
	// and false if the set connection is the same as the current connection
	Select(c *connections.Connection) (bool, error)

	// ExportAsJSON exports all connections as a JSON object (byte slice)
	ExportAsJSON() (connections.ConnectionExport, error)

	// IsLoading returns true if the current selected connection is in read only mode
	IsReadOnly() bool
}

type connectionViewModelImpl struct {
	connRepo     connections.Repository
	settingsVm   SettingsViewModel
	connBindings binding.UntypedList
	// selectedConnection *connection.Connection
	loading        binding.Bool
	connectionsSet *connections.Set
}

var _ ConnectionViewModel = &connectionViewModelImpl{}

func NewConnectionViewModel(
	connRepo connections.Repository,
	settingsVm SettingsViewModel,
) *connectionViewModelImpl {
	c := binding.NewUntypedList()

	vm := &connectionViewModelImpl{
		connRepo:     connRepo,
		settingsVm:   settingsVm,
		connBindings: c,
		loading:      binding.NewBool(),
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
	return c.connBindings
}

func (vm *connectionViewModelImpl) Create(name, accessKey, secretKey, bucket string, options ...connections.ConnectionOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	if err := vm.connectionsSet.Create(name, accessKey, secretKey, bucket, options...); err != nil {
		return fmt.Errorf("error creating connection: %w", err)
	}

	if err := vm.connRepo.Save(ctx, vm.connectionsSet); err != nil {
		// TODO: send to global logging chan
		// vm.errChan <- fmt.Errorf("error saving connection set: %w", err)
		fmt.Printf("error saving connection set: %v", err)
		return fmt.Errorf("error saving connection set after creation: %w", err)
	}

	return nil
}

func (vm *connectionViewModelImpl) Update(c *connections.Connection, options ...connections.ConnectionOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	if err := vm.connectionsSet.Update(c.ID(), options...); err != nil {
		// TODO: send to global logging chan
		// vm.errChan <- fmt.Errorf("error updating connection: %w", err)
		fmt.Printf("error updating connection: %v", err)
		return fmt.Errorf("error updating connection: %w", err)
	}

	if err := vm.connRepo.Save(ctx, vm.connectionsSet); err != nil {
		// TODO: send to global logging chan
		// vm.errChan <- fmt.Errorf("error saving connection set: %w", err)
		fmt.Printf("error saving connection set after update: %v", err)
		return fmt.Errorf("error saving connection set after update: %w", err)
	}
	return nil
}

func (vm *connectionViewModelImpl) Delete(c *connections.Connection) error {
	if err := vm.connectionsSet.Delete(c.ID()); err != nil {
		// TODO: send to global logging chan
		// vm.errChan <- fmt.Errorf("error deleting connection: %w", err)
		fmt.Printf("error deleting connection from set: %v", err)
		return fmt.Errorf("error deleting connection from set: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Delete(ctx, c.ID()); err != nil {
		// TODO: rollback the save event here
		// TODO: send to global logging chan
		// vm.errChan <- fmt.Errorf("error deleting connection: %w", err)
		fmt.Printf("error deleting connection: %v", err)
		return err
	}

	prevConns, err := uiutils.GetUntypedList[*connections.Connection](vm.connBindings)
	if err != nil {
		// TODO: send to global logging chan.
		// vm.errChan <- fmt.Errorf("error getting previous connections: %w", err)
		fmt.Printf("error getting previous connections: %v", err)
		return err
	}

	found := false
	for _, prevConn := range prevConns {
		if prevConn.Is(c) {
			found = vm.connBindings.Remove(prevConn) == nil
		}
	}

	if !found {
		// Is this case possible? If so, we should handle it gracefully.
		return fmt.Errorf("connection with ID %s not found", c.ID())
	}

	return nil
}

func (vm *connectionViewModelImpl) Select(c *connections.Connection) (bool, error) {
	vm.loading.Set(true)
	defer vm.loading.Set(false)

	prevSelected := vm.connectionsSet.Selected()

	if err := vm.connectionsSet.Select(c.ID()); err != nil {
		return false, fmt.Errorf("error selecting connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.connRepo.Save(ctx, vm.connectionsSet); err != nil {
		// TODO: send to global logging chan
		// vm.errChan <- fmt.Errorf("error saving connection set: %w", err)
		if prevSelected != nil {
			// TODO: send event instead calling methods???
			vm.connectionsSet.Select(prevSelected.ID())
		}
		return false, fmt.Errorf("error saving connection set after selection: %w", err)
	}

	if err := vm.updateBinding(c); err != nil {
		if prevSelected != nil {
			newSelected := vm.connectionsSet.Selected()
			vm.connectionsSet.Select(prevSelected.ID())
			if err := vm.connRepo.Save(ctx, vm.connectionsSet); err != nil {
				// TODO: send to global logging chan
				// vm.errChan <- fmt.Errorf("error saving connection set: %w", err)
				vm.connectionsSet.Select(newSelected.ID())
				return false, fmt.Errorf("error saving connection set after selection: %w", err)
			}
		}
		return false, err
	}

	return true, nil
}

func (vm *connectionViewModelImpl) ExportAsJSON() (connections.ConnectionExport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	return vm.connRepo.ExportToJson(ctx)
}

func (vm *connectionViewModelImpl) IsReadOnly() bool {
	if vm.connectionsSet.Selected() == nil {
		return false
	}
	return vm.connectionsSet.Selected().ReadOnly
}

func (vm *connectionViewModelImpl) updateBinding(c *connections.Connection) error {
	allConns, err := uiutils.GetUntypedList[*connections.Connection](vm.connBindings)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error getting previous connections: %w", err)
		fmt.Printf("error listing connections: %v", err)
		return err
	}

	found := false
	for i, conn := range allConns {
		if conn.Is(c) {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := vm.connBindings.SetValue(i, &updatedConn); err != nil {
				// TOOD: send to global logging chan
				// vm.errChan <- fmt.Errorf("error setting selected connection: %w", err)
				fmt.Printf("error updating connection: %v", err)
				return err
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholcerConn := connections.NewEmptyConnection()
			vm.connBindings.Append(placeholcerConn)
			vm.connBindings.Remove(placeholcerConn)
		}
	}

	if !found {
		return errConnNotInBinding
	}

	return nil
}

func (vm *connectionViewModelImpl) loadInitialConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	s, err := vm.connRepo.Get(ctx)
	if err != nil {
		// TOOD: send to global logging chan
		// vm.errChan <- fmt.Errorf("error listing connections: %w", err)
		fmt.Printf("error loading connections: %v", err)
		return err
	}

	vm.connectionsSet = s

	for _, c := range s.Connections() {
		vm.connBindings.Append(c)
	}

	return nil
}
