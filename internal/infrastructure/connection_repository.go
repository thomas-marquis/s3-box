package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/connection"

	"fyne.io/fyne/v2"
	"github.com/google/uuid"
)

const (
	allConnectionsKey = "allConnections"
)

type connectionDTO struct {
	ID        uuid.UUID `json:"id"`
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

func (c *connectionDTO) toConnection() *connection.Connection {
	return &connection.Connection{
		ID:         c.ID,
		Name:       c.Name,
		Server:     c.Server,
		AccessKey:  c.AccessKey,
		SecretKey:  c.SecretKey,
		IsSelected: c.Selected,
		BucketName: c.Buket,
		Region:     c.Region,
		Type:       connection.NewConnectionTypeFromString(c.Type),
		UseTls:     c.UseTls,
		ReadOnly:   c.ReadOnly,
	}
}

func newConnectionDTO(c *connection.Connection) *connectionDTO {
	return &connectionDTO{
		ID:        c.ID,
		Name:      c.Name,
		Server:    c.Server,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
		Selected:  c.IsSelected,
		Buket:     c.BucketName,
		Region:    c.Region,
		Type:      c.Type.String(),
		UseTls:    c.UseTls,
		ReadOnly:  c.ReadOnly,
	}
}

type ConnectionRepositoryImpl struct {
	prefs fyne.Preferences
}

func NewConnectionRepositoryImpl(prefs fyne.Preferences) *ConnectionRepositoryImpl {
	return &ConnectionRepositoryImpl{prefs}
}

var _ connection.Repository = &ConnectionRepositoryImpl{}

func (r *ConnectionRepositoryImpl) ListConnections(ctx context.Context) ([]*connection.Connection, error) {
	dtos, err := r.loadConnectionDTOs()
	if err != nil {
		return nil, fmt.Errorf("ListConnections: %w", err)
	}

	// filter connections to remove those with empty id
	var filteredConnections []*connection.Connection
	for _, dto := range dtos {
		if dto.ID != uuid.Nil {
			filteredConnections = append(filteredConnections, dto.toConnection())
		}
	}

	return filteredConnections, nil
}

func (r *ConnectionRepositoryImpl) SaveConnection(ctx context.Context, c *connection.Connection) error {
	connections, err := r.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("SaveConnection: %w", err)
	}

	var found bool
	for _, conn := range connections {
		if conn.ID == c.ID {
			found = true
			conn.Update(c)
			break
		}
	}

	if !found {
		connections = append(connections, c)
	}

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

func (r *ConnectionRepositoryImpl) DeleteConnection(ctx context.Context, id uuid.UUID) error {
	connections, err := r.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("DeleteConnection: %w", err)
	}

	var connToKeep []*connection.Connection
	var found bool
	for _, c := range connections {
		if c.ID != id {
			connToKeep = append(connToKeep, c)
		} else {
			found = true
		}
	}

	if !found {
		return connection.ErrConnectionNotFound
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

func (r *ConnectionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*connection.Connection, error) {
	connections, err := r.ListConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	for _, c := range connections {
		if c.ID == id {
			return c, nil
		}
	}

	return nil, connection.ErrConnectionNotFound
}

func (r *ConnectionRepositoryImpl) SetSelectedConnection(ctx context.Context, id uuid.UUID) error {
	connections, err := r.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("SetSelectedConnection: %w", err)
	}

	var found bool
	for _, c := range connections {
		if c.ID == id {
			c.IsSelected = true
			found = true
		} else {
			c.IsSelected = false
		}
	}

	if !found {
		return connection.ErrConnectionNotFound
	}

	dtos := make([]*connectionDTO, len(connections))
	for i, c := range connections {
		dtos[i] = newConnectionDTO(c)
	}
	content, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("SetSelectedConnection: %w", err)
	}

	r.prefs.SetString(allConnectionsKey, string(content))

	return nil
}

func (r *ConnectionRepositoryImpl) GetSelectedConnection(ctx context.Context) (*connection.Connection, error) {
	connections, err := r.ListConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetSelectedConnection: %w", err)
	}

	for _, c := range connections {

		if c.IsSelected {
			return c, nil
		}
	}

	return nil, connection.ErrConnectionNotFound
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

func (r *ConnectionRepositoryImpl) ExportToJson(ctx context.Context) (connection.ConnectionExport, error) {
	dtos, err := r.loadConnectionDTOs()
	if err != nil {
		return connection.ConnectionExport{}, fmt.Errorf("ExportToJson: %w", err)
	}
	content, err := json.Marshal(dtos)
	if err != nil {
		return connection.ConnectionExport{}, fmt.Errorf("ExportToJson: %w", err)
	}
	return connection.ConnectionExport{
		JSONData: content,
		Count:    len(dtos),
	}, nil
}
