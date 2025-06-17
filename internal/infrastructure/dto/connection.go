package dto

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
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

type ConnectionsDTO struct {
	connections []*connectionDTO
}

func NewConnectionsDTO(c *connections.Connections) *ConnectionsDTO {
	dtos := make([]*connectionDTO, 0, len(c.Get()))
	selectedID := c.SelectedConnection()

	for _, conn := range c.Get() {
		dto := &connectionDTO{
			ID:        uuid.UUID(conn.ID()),
			Revision:  conn.Revision(),
			Name:      conn.Name(),
			Server:    conn.Server(),
			AccessKey: conn.AccessKey(),
			SecretKey: conn.SecretKey(),
			Buket:     conn.Bucket(),
			Selected:  false,
			Region:    conn.Region(),
			Type:      conn.Provider().String(),
			UseTls:    conn.UseTLS(),
			ReadOnly:  conn.ReadOnly(),
		}
		if selectedID != nil && selectedID.Is(conn) {
			dto.Selected = true
		}
		dtos = append(dtos, dto)
	}

	return &ConnectionsDTO{
		connections: dtos,
	}
}

func (c *ConnectionsDTO) ToConnections() *connections.Connections {
	conns := connections.New()
	nilDI := connections.ConnectionID(uuid.Nil)
	selectedID := nilDI
	for _, dto := range c.connections {
		connID := connections.ConnectionID(dto.ID)
		conns.NewConnection(
			dto.Name, dto.AccessKey, dto.SecretKey, dto.Buket,
			connections.WithRevision(dto.Revision),
			connections.WithUseTLS(dto.UseTls),
			connections.WithID(connID),
			connections.WithReadOnlyOption(dto.ReadOnly),
			connections.AsS3Like(dto.Server, dto.UseTls),
			connections.AsAWS(dto.Region),
		)
		if dto.Selected {
			selectedID = connID
		}
	}
	if selectedID != nilDI {
		conns.Select(selectedID)
	}
	return conns
}

func (c *ConnectionsDTO) Serialize() ([]byte, error) {
	content, err := json.Marshal(c.connections)
	if err != nil {
		return nil, err
	}
	return content, nil
}
