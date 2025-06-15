package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/connections"

	"fyne.io/fyne/v2"
	"github.com/google/uuid"
)

const (
	allConnectionsKey = "allConnections"
)

type connectionDTO struct {
	ID        uuid.UUID `json:"id"`
	Revision  int       `json:"revision,omitempty"`
	Name      string    `json:"name"`
	Server    string    `json:"server"`
	AccessKey string    `json:"accessKey"`
	SecretKey string    `json:"secretKey"`
	Buket     string    `json:"bucket"`
	Selected  bool      `json:"selected,omitempty"`
	Region    string    `json:"region,omitempty"`
	Type      string    `json:"type,omitempty"`
	UseTls    bool      `json:"useTls,omitempty"`
	ReadOnly  bool      `json:"readOnly,omitempty"`
}

func (c *connectionDTO) toConnection() *connections.Connection {
	return connections.New(
		c.Name, c.AccessKey, c.SecretKey, c.Buket,
		connections.WithRevision(c.Revision),
		connections.WithSelected(c.Selected),
		connections.WithUseTLS(c.UseTls),
		connections.WithID(c.ID),
		connections.WithReadOnlyOption(c.ReadOnly),
		connections.AsS3LikeConnection(c.Server, c.UseTls),
		connections.AsAWSConnection(c.Region),
	)
}

func newConnectionDTO(c *connections.Connection) *connectionDTO {
	return &connectionDTO{
		ID:        c.ID(),
		Name:      c.Name,
		Server:    c.Server,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
		Selected:  c.Selected(),
		Buket:     c.BucketName,
		Region:    c.Region,
		Type:      c.Type.String(),
		UseTls:    c.UseTls,
		ReadOnly:  c.ReadOnly,
		Revision:  c.Revision(),
	}
}

type ConnectionRepositoryImpl struct {
	prefs fyne.Preferences
}

func NewConnectionRepositoryImpl(prefs fyne.Preferences) *ConnectionRepositoryImpl {
	return &ConnectionRepositoryImpl{prefs}
}

var _ connections.Repository = &ConnectionRepositoryImpl{}

func (r *ConnectionRepositoryImpl) List(ctx context.Context) ([]*connections.Connection, error) {
	dtos, err := r.loadConnectionDTOs()
	if err != nil {
		return nil, fmt.Errorf("ListConnections: %w", err)
	}

	// filter connections to remove those with empty id
	var filteredConnections []*connections.Connection
	for _, dto := range dtos {
		if dto.ID != uuid.Nil {
			filteredConnections = append(filteredConnections, dto.toConnection())
		}
	}

	return filteredConnections, nil
}

func (r *ConnectionRepositoryImpl) Get(ctx context.Context) (*connections.Set, error) {
	conns, err := r.listConnections()
	if err != nil {
		return nil, fmt.Errorf("GetConnections: %w", err)
	}
	return connections.NewSet(connections.WithConnections(conns)), nil
}

func (r *ConnectionRepositoryImpl) Save(ctx context.Context, s *connections.Set) error {
	connections := s.Connections()
	// prevConns, err := r.List(ctx)
	// connections, err := r.List(ctx)
	// if err != nil {
	// 	return fmt.Errorf("SaveConnection: %w", err)
	// }
	//
	// var found bool
	// for _, conn := range prevConns {
	// 	for _, c := range updatedConns {
	// 		if conn.Is(c) {
	// 			found = true
	// 			conn.Update(c)
	// 			break
	// 		}
	// 	}
	// }
	//
	// if !found {
	// 	connections = append(connections, c)
	// }

	dtos := make([]*connectionDTO, len(connections))
	for i, c := range connections {
		dtos[i] = newConnectionDTO(c)
	}
	content, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("SaveConnection: %w", err)
	}

	r.prefs.SetString(allConnectionsKey, string(content))

	return nil
}

func (r *ConnectionRepositoryImpl) listConnections() ([]*connections.Connection, error) {
	dtos, err := r.loadConnectionDTOs()
	if err != nil {
		return nil, fmt.Errorf("ListConnections: %w", err)
	}

	// filter connections to remove those with empty id
	var filteredConnections []*connections.Connection
	for _, dto := range dtos {
		if dto.ID != uuid.Nil {
			filteredConnections = append(filteredConnections, dto.toConnection())
		}
	}

	return filteredConnections, nil
}

func (r *ConnectionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	conns, err := r.List(ctx)
	if err != nil {
		return fmt.Errorf("DeleteConnection: %w", err)
	}

	var connToKeep []*connections.Connection
	var found bool
	for _, c := range conns {
		if c.ID() != id {
			connToKeep = append(connToKeep, c)
		} else {
			found = true
		}
	}

	if !found {
		return connections.ErrConnectionNotFound
	}

	dtos := make([]*connectionDTO, len(connToKeep))
	for i, c := range connToKeep {
		dtos[i] = newConnectionDTO(c)
	}
	content, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("DeleteConnection: %w", err)
	}

	r.prefs.SetString(allConnectionsKey, string(content))

	return nil
}

func (r *ConnectionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*connections.Connection, error) {
	conns, err := r.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	for _, c := range conns {
		if c.ID() == id {
			return c, nil
		}
	}

	return nil, connections.ErrConnectionNotFound
}

func (r *ConnectionRepositoryImpl) SetSelected(ctx context.Context, id uuid.UUID) error {
	conns, err := r.List(ctx)
	if err != nil {
		return fmt.Errorf("SetSelectedConnection: %w", err)
	}

	var found bool
	for _, c := range conns {
		if c.ID() == id {
			c.Select()
			found = true
		} else {
			c.Unselect()
		}
	}

	if !found {
		return connections.ErrConnectionNotFound
	}

	dtos := make([]*connectionDTO, len(conns))
	for i, c := range conns {
		dtos[i] = newConnectionDTO(c)
	}
	content, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("SetSelectedConnection: %w", err)
	}

	r.prefs.SetString(allConnectionsKey, string(content))

	return nil
}

func (r *ConnectionRepositoryImpl) GetSelected(ctx context.Context) (*connections.Connection, error) {
	conns, err := r.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetSelectedConnection: %w", err)
	}

	for _, c := range conns {

		if c.Selected() {
			return c, nil
		}
	}

	return nil, connections.ErrConnectionNotFound
}

// loadConnectionDTOs loads the connectionDTOs directly from preferences
func (r *ConnectionRepositoryImpl) loadConnectionDTOs() ([]*connectionDTO, error) {
	content := r.prefs.String(allConnectionsKey)
	if content == "" || content == "null" {
		return []*connectionDTO{}, nil
	}
	dtos, err := fromJson[[]*connectionDTO](content)
	if err != nil {
		return nil, err
	}
	return dtos, nil
}

func (r *ConnectionRepositoryImpl) ExportToJson(ctx context.Context) (connections.ConnectionExport, error) {
	dtos, err := r.loadConnectionDTOs()
	if err != nil {
		return connections.ConnectionExport{}, fmt.Errorf("ExportToJson: %w", err)
	}
	content, err := json.Marshal(dtos)
	if err != nil {
		return connections.ConnectionExport{}, fmt.Errorf("ExportToJson: %w", err)
	}
	return connections.ConnectionExport{
		JSONData: content,
		Count:    len(dtos),
	}, nil
}
