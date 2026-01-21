package dto_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/dto"
)

func TestNewConnectionsDTO(t *testing.T) {
	t.Run("should create DTO from an empty deck", func(t *testing.T) {
		// Given
		deck := connection_deck.New()

		// When
		d := dto.NewConnectionsDTO(deck)

		// Then
		assert.NotNil(t, d)
		data, err := json.Marshal(d)
		assert.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("should create DTO with multiple connections and selection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		c1 := deck.New("conn 1", "ak1", "sk1", "b1", connection_deck.AsAWS("us-east-1")).Connection()
		c2 := deck.New("conn 2", "ak2", "sk2", "b2", connection_deck.AsS3Like("http://localhost:9000", true)).Connection()
		_, _ = deck.Select(c2.ID())

		// When
		d := dto.NewConnectionsDTO(deck)

		// Then
		assert.NotNil(t, d)
		data, err := json.Marshal(d)
		assert.NoError(t, err)

		expected := fmt.Sprintf(`[
			{
				"id": "%s",
				"name": "conn 1",
				"server": "",
				"accessKey": "ak1",
				"secretKey": "sk1",
				"bucket": "b1",
				"type": "aws",
				"region": "us-east-1",
				"useTls": true
			},
			{
				"id": "%s",
				"name": "conn 2",
				"server": "http://localhost:9000",
				"accessKey": "ak2",
				"secretKey": "sk2",
				"bucket": "b2",
				"selected": true,
				"type": "s3-like",
				"useTls": true
			}
		]`, c1.ID(), c2.ID())

		assert.JSONEq(t, expected, string(data))
	})
}

func TestNewConnectionsDTOFromJSON(t *testing.T) {
	t.Run("should create DTO from valid JSON", func(t *testing.T) {
		// Given
		content := `
[
  {
    "id": "00000000-0000-0000-0000-000000000001",
    "revision": 1,
    "name": "test connection",
    "server": "http://localhost:9000",
    "accessKey": "ak",
    "secretKey": "sk",
    "bucket": "b1",
    "selected": true,
    "type": "s3-like"
  }
]
`

		// When
		d, err := dto.NewConnectionsDTOFromJSON([]byte(content))

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, d)

		data, err := json.Marshal(d)
		assert.NoError(t, err)
		assert.JSONEq(t, content, string(data))
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		// Given
		content := `invalid json`

		// When
		d, err := dto.NewConnectionsDTOFromJSON([]byte(content))

		// Then
		assert.Error(t, err)
		assert.Nil(t, d)
	})
}

func TestConnectionsDTO_ToConnections(t *testing.T) {
	t.Run("should convert DTO back to deck with correct attributes", func(t *testing.T) {
		// Given
		id1 := uuid.New()
		id2 := uuid.New()
		content := fmt.Sprintf(`[
			{
				"id": "%s",
				"name": "aws conn",
				"accessKey": "ak1",
				"secretKey": "sk1",
				"bucket": "b1",
				"type": "aws",
				"region": "eu-west-1",
				"revision": 5,
				"readOnly": true
			},
			{
				"id": "%s",
				"name": "s3 conn",
				"accessKey": "ak2",
				"secretKey": "sk2",
				"bucket": "b2",
				"type": "s3-like",
				"server": "http://minio:9000",
				"useTls": true,
				"selected": true
			}
		]`, id1, id2)
		d, _ := dto.NewConnectionsDTOFromJSON([]byte(content))

		// When
		deck := d.ToConnections()

		// Then
		require.Len(t, deck.Get(), 2)

		c1, err := deck.GetByID(connection_deck.ConnectionID(id1))
		assert.NoError(t, err)
		assert.Equal(t, "aws conn", c1.Name())
		assert.Equal(t, "eu-west-1", c1.Region())
		assert.Equal(t, 5, c1.Revision())
		assert.True(t, c1.ReadOnly())

		c2, err := deck.GetByID(connection_deck.ConnectionID(id2))
		assert.NoError(t, err)
		assert.Equal(t, "s3 conn", c2.Name())
		assert.Equal(t, "http://minio:9000", c2.Server())
		assert.True(t, c2.IsTLSActivated())
		assert.Equal(t, c2, deck.SelectedConnection())
	})

	t.Run("should skip connections with nil ID", func(t *testing.T) {
		// Given
		content := `[
			{
				"id": "00000000-0000-0000-0000-000000000000",
				"name": "nil id"
			}
		]`
		d, _ := dto.NewConnectionsDTOFromJSON([]byte(content))

		// When
		deck := d.ToConnections()

		// Then
		assert.Len(t, deck.Get(), 0)
	})
}

func TestConnectionsDTO_MarshalJSON(t *testing.T) {
	t.Run("should marshal connections correctly", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		c1 := deck.New("conn 1", "ak1", "sk1", "b1", connection_deck.AsAWS("us-east-1")).Connection()
		d := dto.NewConnectionsDTO(deck)

		// When
		data, err := d.MarshalJSON()

		// Then
		assert.NoError(t, err)
		expected := `[
			{
				"id":"` + uuid.UUID(c1.ID()).String() + `",` + `
			    "name":"conn 1",
				"server":"",
				"accessKey":"ak1",
				"secretKey":"sk1",
				"bucket":"b1",
				"type":"aws",
				"region":"us-east-1",
				"useTls":true
			}
		]`
		assert.JSONEq(t, expected, string(data))
	})
}
