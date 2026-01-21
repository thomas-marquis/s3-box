package dto

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

type connectionDTO struct {
	ID        uuid.UUID `json:"id"`
	Revision  int       `json:"revision,omitempty"`
	Name      string    `json:"name"`
	Server    string    `json:"server,omitempty"`
	AccessKey string    `json:"accessKey"`
	SecretKey string    `json:"secretKey"`
	Bucket    string    `json:"bucket"`
	Selected  bool      `json:"selected,omitempty"`
	Region    string    `json:"region,omitempty"`
	Type      string    `json:"type,omitempty"`
	UseTls    bool      `json:"useTls,omitempty"`
	ReadOnly  bool      `json:"readOnly,omitempty"`
}

type ConnectionsDTO struct {
	connections []*connectionDTO
}

func NewConnectionsDTO(c *connection_deck.Deck) *ConnectionsDTO {
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
			Bucket:    conn.Bucket(),
			Selected:  false,
			Region:    conn.Region(),
			Type:      conn.Provider().String(),
			UseTls:    conn.IsTLSActivated(),
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

func NewConnectionsDTOFromJSON(content []byte) (*ConnectionsDTO, error) {
	var dtos []*connectionDTO
	if err := json.Unmarshal(content, &dtos); err != nil {
		return nil, err
	}
	return &ConnectionsDTO{connections: dtos}, nil
}

func (c *ConnectionsDTO) ToConnections() *connection_deck.Deck {
	conns := connection_deck.New()
	nilID := connection_deck.ConnectionID(uuid.Nil)
	selectedID := nilID
	for _, dto := range c.connections {
		if dto.ID == uuid.Nil {
			continue
		}
		connID := connection_deck.ConnectionID(dto.ID)
		evt := conns.New(
			dto.Name, dto.AccessKey, dto.SecretKey, dto.Bucket,
			connection_deck.WithRevision(dto.Revision),
			connection_deck.WithUseTLS(dto.UseTls),
			connection_deck.WithID(connID),
			connection_deck.WithReadOnlyOption(dto.ReadOnly),
		)
		newConn := evt.Connection()
		if dto.Type == "aws" {
			connection_deck.AsAWS(dto.Region)(newConn)
		} else if dto.Type == "s3-like" {
			connection_deck.AsS3Like(dto.Server, dto.UseTls)(newConn)
		}
		if dto.Selected {
			selectedID = connID
		}
	}
	if selectedID != nilID {
		conns.Select(selectedID) //nolint:errcheck
	}
	return conns
}

func (c *ConnectionsDTO) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.connections)
}
