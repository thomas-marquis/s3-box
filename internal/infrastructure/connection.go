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


type ConnectionRepositoryImpl struct {
	prefs fyne.Preferences
}

func NewConnectionRepositoryImpl(prefs fyne.Preferences) *ConnectionRepositoryImpl {
	return &ConnectionRepositoryImpl{prefs}
}

var _ connection.Repository = &ConnectionRepositoryImpl{}

func (r *ConnectionRepositoryImpl) ListConnections(ctx context.Context) ([]*connection.Connection, error) {
	content := r.prefs.String(allConnectionsKey)
	if content == "" || content == "null" {
		return []*connection.Connection{}, nil
	}

    connections, err := fromJson[[]*connection.Connection](content)
    if err != nil {
        return nil, fmt.Errorf("ListConnections: %w", err)
    }

    // filter connections to remove those with empty id
    var filteredConnections []*connection.Connection
    for _, c := range connections {
        if c.ID != uuid.Nil {
            filteredConnections = append(filteredConnections, c)
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
            conn.Name = c.Name
            conn.Server = c.Server
            conn.AccessKey = c.AccessKey
            conn.SecretKey = c.SecretKey
            conn.BucketName = c.BucketName
            conn.UseTls = c.UseTls
            conn.IsSelected = c.IsSelected
            conn.Region = c.Region
            break
        }
    }

    if !found {
        connections = append(connections, c)
    }

    content, err := json.Marshal(connections)
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

    content, err := json.Marshal(connToKeep)
    if err != nil {
        return fmt.Errorf("DeleteConnection: %w", err)
    }

    r.prefs.SetString(allConnectionsKey, string(content))

    return nil
}

func (r *ConnectionRepositoryImpl) GetConnection(ctx context.Context, name string) (*connection.Connection, error) {
    connections, err := r.ListConnections(ctx)
    if err != nil {
        return nil, fmt.Errorf("GetConnection: %w", err)
    }

    for _, c := range connections {
        if c.Name == name {
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

    content, err := json.Marshal(connections)
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

func fromJson[T any](content string) (T, error) {
    var structType T
	err := json.Unmarshal([]byte(content), &structType)
    if err != nil {
        return structType, fmt.Errorf("fromJson: %w", err)
    }
	return structType, nil
}

