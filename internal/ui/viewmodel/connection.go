package viewmodel

import (
	"context"

	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

type ConnectionViewModel struct {
	explorerRepo       explorer.Repository
	connRepo           connection.Repository
	dirSvc             *explorer.DirectoryService
	connections        binding.UntypedList
	loading            binding.Bool
	lastSaveDir        fyne.ListableURI
	lastUploadDir      fyne.ListableURI
	selectedConnection *connection.Connection
}

func NewConnectionViewModel(explorerRepo explorer.Repository, dirScv *explorer.DirectoryService, connRepo connection.Repository) *ConnectionViewModel {
	c := binding.NewUntypedList()

	v := &ConnectionViewModel{
		explorerRepo: explorerRepo,
		dirSvc:       dirScv,
		connections:  c,
		connRepo:     connRepo,
		loading:      binding.NewBool(),
	}

	if err := v.RefreshConnections(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	currentConn, err := connRepo.GetSelectedConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		panic(err)
	}
	if currentConn != nil {
		v.selectedConnection = currentConn
	}

	return v
}

func (v *ConnectionViewModel) Loading() binding.Bool {
	return v.loading
}

func (v *ConnectionViewModel) StartLoading() {
	v.loading.Set(true)
}

func (v *ConnectionViewModel) StopLoading() {
	v.loading.Set(false)
}

func (v *ConnectionViewModel) Connections() binding.UntypedList {
	return v.connections
}

func (v *ConnectionViewModel) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conns, err := v.connRepo.ListConnections(ctx)
	if err != nil {
		return err
	}

	prevConns, err := v.connections.Get()
	if err != nil {
		return err
	}
	for _, c := range prevConns {
		v.connections.Remove(c)
	}

	for _, c := range conns {
		v.connections.Append(c)
	}

	return nil
}

func (v *ConnectionViewModel) SaveConnection(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := v.connRepo.SaveConnection(ctx, c); err != nil {
		return err
	}

	return v.RefreshConnections()
}

func (v *ConnectionViewModel) DeleteConenction(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := v.connRepo.DeleteConnection(ctx, c.ID); err != nil {
		return err
	}

	return v.RefreshConnections()
}

func (v *ConnectionViewModel) SelectedConnection() *connection.Connection {
	return v.selectedConnection
}

func (v *ConnectionViewModel) SelectConnection(c *connection.Connection) error {
	v.loading.Set(true)
	defer v.loading.Set(false)

	prevConn := v.selectedConnection
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	if err := v.connRepo.SetSelectedConnection(ctx, c.ID); err != nil {
		cancel()
		return err
	}
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := v.explorerRepo.SetConnection(ctx, c); err != nil {
		if err := v.connRepo.SetSelectedConnection(ctx, prevConn.ID); err != nil {
			return err
		}
		return err
	}

	v.selectedConnection = c
	return nil
}
