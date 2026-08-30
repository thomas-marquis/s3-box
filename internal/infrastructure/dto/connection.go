package dto

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/u"
)

type ConnectionDTO struct {
	ID        string `json:"id"`
	Revision  int    `json:"revision,omitempty"`
	Name      string `json:"name"`
	Server    string `json:"server,omitempty"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Bucket    string `json:"bucket"`
	Selected  bool   `json:"selected,omitempty"`
	Region    string `json:"region,omitempty"`
	Type      string `json:"type,omitempty"`
	UseTls    bool   `json:"useTls,omitempty"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

type ConnectionsDTO struct {
	Connections []*ConnectionDTO `json:"connections"`
}

func NewConnectionsDTO(c *connection_deck.Deck) *ConnectionsDTO {
	dtos := make([]*ConnectionDTO, 0, len(c.Get()))
	selectedID := c.SelectedConnection()

	for _, conn := range c.Get() {
		dto := &ConnectionDTO{
			ID:        conn.ID().String(),
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
		Connections: dtos,
	}
}

func NewConnectionsDTOFromList(conns []*connection_deck.Connection) *ConnectionsDTO {
	dtos := make([]*ConnectionDTO, 0, len(conns))
	for _, conn := range conns {
		dto := &ConnectionDTO{
			ID:        conn.ID().String(),
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
		dtos = append(dtos, dto)
	}
	return &ConnectionsDTO{
		Connections: dtos,
	}
}

func NewConnectionsDTOFromJSON(content []byte) (*ConnectionsDTO, error) {
	var dtos []*ConnectionDTO
	if err := json.Unmarshal(content, &dtos); err != nil {
		return nil, err
	}
	return &ConnectionsDTO{Connections: dtos}, nil
}

func (c *ConnectionsDTO) ToConnections() *connection_deck.Deck {
	conns := connection_deck.New()
	nilID := connection_deck.ConnectionID(uuid.Nil)
	selectedID := nilID
	for _, dto := range c.Connections {
		if dto.ID == "" {
			continue
		}

		id, err := uuid.Parse(dto.ID)
		if err != nil || id == uuid.Nil {
			continue
		}

		connID := connection_deck.ConnectionID(id)
		evt := conns.New(
			dto.Name, dto.AccessKey, dto.SecretKey, dto.Bucket,
			connection_deck.WithRevision(dto.Revision),
			connection_deck.WithUseTLS(dto.UseTls),
			connection_deck.WithID(connID),
			connection_deck.WithReadOnlyOption(dto.ReadOnly),
		)
		newConn := evt.Payload().(connection_deck.CreateConnectionTriggered).Connection()
		switch dto.Type {
		case "aws":
			connection_deck.AsAWS(dto.Region)(newConn)
		case "s3-like":
			connection_deck.AsS3Like(dto.Server, dto.UseTls)(newConn)
		}
		if dto.Selected {
			selectedID = connID
		}
	}
	if selectedID != nilID {
		u.SkipV(conns.Select(selectedID))
	}
	return conns
}

func (c *ConnectionsDTO) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Connections)
}
