package viewmodel

import (
	"context"

	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

type ConnectionViewModel struct {
	explorerRepo  explorer.Repository
	connRepo      connection.Repository
	dirSvc        *explorer.DirectoryService
	connections   binding.UntypedList
	state         AppState
	loading       binding.Bool
	lastSaveDir   fyne.ListableURI
	lastUploadDir fyne.ListableURI
}

func NewConnectionViewModel(explorerRepo explorer.Repository, dirScv *explorer.DirectoryService, connRepo connection.Repository) *ConnectionViewModel {
	c := binding.NewUntypedList()

	vm := &ConnectionViewModel{
		explorerRepo: explorerRepo,
		dirSvc:       dirScv,
		connections:  c,
		connRepo:     connRepo,
		state:        NewAppState(),
		loading:      binding.NewBool(),
	}

	if err := vm.RefreshConnections(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	currentConn, err := connRepo.GetSelectedConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		panic(err)
	}
	if currentConn != nil {
		vm.state.SelectedConnection = currentConn
	}

	return vm
}

func (vm *ConnectionViewModel) Loading() binding.Bool {
	return vm.loading
}

func (vm *ConnectionViewModel) StartLoading() {
	vm.loading.Set(true)
}

func (vm *ConnectionViewModel) StopLoading() {
	vm.loading.Set(false)
}

func (vm *ConnectionViewModel) Connections() binding.UntypedList {
	return vm.connections
}

func (vm *ConnectionViewModel) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conns, err := vm.connRepo.ListConnections(ctx)
	if err != nil {
		return err
	}

	prevConns, err := vm.connections.Get()
	if err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.connRepo.SaveConnection(ctx, c); err != nil {
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ConnectionViewModel) DeleteConenction(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.connRepo.DeleteConnection(ctx, c.ID); err != nil {
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ConnectionViewModel) SelectedConnection() *connection.Connection {
	return vm.state.SelectedConnection
}

func (vm *ConnectionViewModel) SelectConnection(c *connection.Connection) error {
	vm.loading.Set(true)
	prevConn := vm.state.SelectedConnection

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	if err := vm.connRepo.SetSelectedConnection(ctx, c.ID); err != nil {
		cancel()
		return err
	}
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.explorerRepo.SetConnection(ctx, c); err != nil {
		if err := vm.connRepo.SetSelectedConnection(ctx, prevConn.ID); err != nil {
			return err
		}
		return err
	}

	if c != prevConn {
		vm.resetTreeContent()
		explorer.RootDir.IsLoaded = false // TODO: crado, crecréer un rootdir plutôt
		explorer.RootDir.SubDirectories = make([]*explorer.Directory, 0)
		explorer.RootDir.Files = make([]*explorer.RemoteFile, 0)
		if err := vm.tree.Append("", explorer.RootDir.Path(), explorer.RootDir); err != nil {
			return err
		}
		if err := vm.ExpandDir(explorer.RootDir); err != nil {
			return err
		}
	}

	vm.state.SelectedConnection = c
	vm.loading.Set(false)
	return nil
}
