package connection_deck_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

func TestDeck_New(t *testing.T) {
	t.Run("should create a new connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()

		// When
		res := deck.New("connection 1", "accesskey", "secretkey", "myBucket")

		// Then
		assert.NotNil(t, res)
		assert.Equal(t, deck, res.Deck())
		assert.Equal(t, "connection 1", res.Connection().Name())
		assert.Equal(t, "accesskey", res.Connection().AccessKey())
		assert.Equal(t, "secretkey", res.Connection().SecretKey())
		assert.Equal(t, "myBucket", res.Connection().Bucket())
	})
}

func TestDeck_GetByID(t *testing.T) {
	t.Run("should return a connection when ID exists", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		event := deck.New("connection 1", "accesskey", "secretkey", "myBucket")
		conn := event.Connection()

		// When
		res, err := deck.GetByID(conn.ID())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, conn, res)
	})

	t.Run("should return ErrNotFound when ID does not exist", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		randomID := connection_deck.NewConnectionID()

		// When
		res, err := deck.GetByID(randomID)

		// Then
		assert.ErrorIs(t, err, connection_deck.ErrNotFound)
		assert.Nil(t, res)
	})
}

func TestDeck_Select(t *testing.T) {
	t.Run("should select a connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()

		// When
		event, err := deck.Select(conn1.ID())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, conn1, deck.SelectedConnection())
		assert.Equal(t, conn1, event.Connection())
		assert.Nil(t, event.Previous())
	})

	t.Run("should update selection and return previous connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
		conn2 := deck.New("conn 2", "ak", "sk", "b2").Connection()
		_, _ = deck.Select(conn1.ID())

		// When
		event, err := deck.Select(conn2.ID())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, conn2, deck.SelectedConnection())
		assert.Equal(t, conn2, event.Connection())
		assert.Equal(t, conn1, event.Previous())
	})

	t.Run("should return ErrNotFound when selecting non-existent connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		randomID := connection_deck.NewConnectionID()

		// When
		_, err := deck.Select(randomID)

		// Then
		assert.ErrorIs(t, err, connection_deck.ErrNotFound)
	})
}

func TestDeck_RemoveAConnection(t *testing.T) {
	t.Run("should remove a connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
		conn2 := deck.New("conn 2", "ak", "sk", "b2").Connection()

		// When
		event, err := deck.RemoveAConnection(conn1.ID())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 1, len(deck.Get()))
		assert.Equal(t, conn1, event.Connection())
		assert.Equal(t, 0, event.RemovedIndex())
		assert.False(t, event.WasSelected())
		assert.NotContains(t, deck.Get(), conn1)
		assert.Contains(t, deck.Get(), conn2)
	})

	t.Run("should reset selection if removed connection was selected", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
		_, _ = deck.Select(conn1.ID())

		// When
		event, err := deck.RemoveAConnection(conn1.ID())

		// Then
		assert.NoError(t, err)
		assert.Nil(t, deck.SelectedConnection())
		assert.True(t, event.WasSelected())
	})

	t.Run("should return ErrNotFound when removing non-existent connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		randomID := connection_deck.NewConnectionID()

		// When
		_, err := deck.RemoveAConnection(randomID)

		// Then
		assert.ErrorIs(t, err, connection_deck.ErrNotFound)
	})
}

func TestDeck_Get(t *testing.T) {
	t.Run("should return all connections", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		deck.New("conn 1", "ak", "sk", "b1")
		deck.New("conn 2", "ak", "sk", "b2")

		// When
		conns := deck.Get()

		// Then
		assert.Equal(t, 2, len(conns))
		assert.Equal(t, "conn 1", conns[0].Name())
		assert.Equal(t, "conn 2", conns[1].Name())
	})
}

func TestDeck_Update(t *testing.T) {
	t.Run("should update a connection and increment revision using various options", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
		customID := connection_deck.NewConnectionID()

		// When
		_, err := deck.Update(conn1.ID(),
			connection_deck.WithName("new name"),
			connection_deck.WithCredentials("new ak", "new sk"),
			connection_deck.WithBucket("new bucket"),
			connection_deck.AsS3Like("http://localhost:9000", false),
			connection_deck.WithReadOnlyOption(true),
			connection_deck.WithRevision(10),
			connection_deck.WithID(customID),
		)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "new name", conn1.Name())
		assert.Equal(t, "new ak", conn1.AccessKey())
		assert.Equal(t, "new sk", conn1.SecretKey())
		assert.Equal(t, "new bucket", conn1.Bucket())
		assert.Equal(t, "http://localhost:9000", conn1.Server())
		assert.False(t, conn1.IsTLSActivated())
		assert.True(t, conn1.ReadOnly())
		assert.Equal(t, 12, conn1.Revision())
		assert.Equal(t, customID, conn1.ID())
	})

	t.Run("should update a connection as AWS", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()

		// When
		_, err := deck.Update(conn1.ID(), connection_deck.AsAWS("eu-west-1"))

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "eu-west-1", conn1.Region())
		assert.True(t, conn1.IsTLSActivated())
	})

	t.Run("should update a connection with TLS", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn1 := deck.New("conn 1", "ak", "sk", "b1", connection_deck.AsS3Like("srv", false)).Connection()

		// When
		_, err := deck.Update(conn1.ID(), connection_deck.WithUseTLS(true))

		// Then
		assert.NoError(t, err)
		assert.True(t, conn1.IsTLSActivated())
	})

	t.Run("should return ErrNotFound when updating non-existent connection", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		randomID := connection_deck.NewConnectionID()

		// When
		_, err := deck.Update(randomID, connection_deck.WithName("new name"))

		// Then
		assert.ErrorIs(t, err, connection_deck.ErrNotFound)
	})
}

func TestDeck_Notify(t *testing.T) {
	t.Run("CreateFailureEvent", func(t *testing.T) {
		t.Run("should remove the connection from the deck", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn := deck.New("conn 1", "ak", "sk", "b1").Connection()
			require.Len(t, deck.Get(), 1)

			// When
			deck.Notify(connection_deck.NewCreateFailureEvent(assert.AnError, conn))

			// Then
			assert.Len(t, deck.Get(), 0)
		})

		t.Run("should do nothing if the connection is not in the deck", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn := connection_deck.New().New("conn 1", "ak", "sk", "b1").Connection()
			require.Len(t, deck.Get(), 0)

			// When
			deck.Notify(connection_deck.NewCreateFailureEvent(assert.AnError, conn))

			// Then
			assert.Len(t, deck.Get(), 0)
		})

		t.Run("should do nothing if the connection is nil", func(t *testing.T) {
			// Given
			deck := connection_deck.New()

			// When
			deck.Notify(connection_deck.NewCreateFailureEvent(assert.AnError, nil))

			// Then
			assert.Len(t, deck.Get(), 0)
		})
	})

	t.Run("SelectFailureEvent", func(t *testing.T) {
		t.Run("should restore the previous selection", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
			conn2 := deck.New("conn 2", "ak", "sk", "b2").Connection()
			_, _ = deck.Select(conn1.ID())
			_, _ = deck.Select(conn2.ID())
			require.Equal(t, conn2, deck.SelectedConnection())

			// When
			deck.Notify(connection_deck.NewSelectFailureEvent(assert.AnError, conn1))

			// Then
			assert.Equal(t, conn1, deck.SelectedConnection())
		})

		t.Run("should do nothing if previous connection is nil", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
			_, _ = deck.Select(conn1.ID())
			require.Equal(t, conn1, deck.SelectedConnection())

			// When
			deck.Notify(connection_deck.NewSelectFailureEvent(assert.AnError, nil))

			// Then
			assert.Equal(t, conn1, deck.SelectedConnection())
		})
	})

	t.Run("RemoveFailureEvent", func(t *testing.T) {
		t.Run("should restore the removed connection and its selection status", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn1 := deck.New("conn 1", "ak", "sk", "b1").Connection()
			conn2 := deck.New("conn 2", "ak", "sk", "b2").Connection()
			_, _ = deck.Select(conn1.ID())

			removeEvent, _ := deck.RemoveAConnection(conn1.ID())
			require.Len(t, deck.Get(), 1)
			require.Nil(t, deck.SelectedConnection())

			// When
			failureEvent := connection_deck.NewRemoveFailureEvent(
				assert.AnError,
				removeEvent.RemovedIndex(),
				removeEvent.WasSelected(),
				removeEvent.Connection(),
			)
			deck.Notify(failureEvent)

			// Then
			assert.Len(t, deck.Get(), 2)
			assert.Equal(t, conn1, deck.Get()[0]) // restored at index 0
			assert.Equal(t, conn2, deck.Get()[1])
			assert.Equal(t, conn1, deck.SelectedConnection())
		})

		t.Run("should do nothing if connection is nil", func(t *testing.T) {
			// Given
			deck := connection_deck.New()

			// When
			deck.Notify(connection_deck.NewRemoveFailureEvent(assert.AnError, 0, false, nil))

			// Then
			assert.Len(t, deck.Get(), 0)
		})

		t.Run("should handle out of bounds index by appending at the end", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn := connection_deck.New().New("conn 1", "ak", "sk", "b1").Connection()

			// When
			deck.Notify(connection_deck.NewRemoveFailureEvent(assert.AnError, 5, false, conn))

			// Then
			assert.Len(t, deck.Get(), 1)
			assert.Equal(t, conn, deck.Get()[0])
		})

		t.Run("should handle negative index by prepending at the beginning", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn := connection_deck.New().New("conn 1", "ak", "sk", "b1").Connection()

			// When
			deck.Notify(connection_deck.NewRemoveFailureEvent(assert.AnError, -1, false, conn))

			// Then
			assert.Len(t, deck.Get(), 1)
			assert.Equal(t, conn, deck.Get()[0])
		})
	})

	t.Run("UpdateFailureEvent", func(t *testing.T) {
		t.Run("should roll back the connection to its previous state", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn := deck.New("conn 1", "ak", "sk", "b1").Connection()
			updateEvent, _ := deck.Update(conn.ID(), connection_deck.WithName("new name"))
			require.Equal(t, "new name", conn.Name())

			// When
			deck.Notify(connection_deck.NewUpdateFailureEvent(assert.AnError, updateEvent.Previous()))

			// Then
			assert.Equal(t, "conn 1", deck.Get()[0].Name())
			assert.Equal(t, updateEvent.Previous().Revision(), deck.Get()[0].Revision())
		})

		t.Run("should do nothing if connection is not in the deck", func(t *testing.T) {
			// Given
			deck := connection_deck.New()
			conn := connection_deck.New().New("conn 1", "ak", "sk", "b1").Connection()

			// When
			deck.Notify(connection_deck.NewUpdateFailureEvent(assert.AnError, conn))

			// Then
			assert.Len(t, deck.Get(), 0)
		})

		t.Run("should do nothing if connection is nil", func(t *testing.T) {
			// Given
			deck := connection_deck.New()

			// When
			deck.Notify(connection_deck.NewUpdateFailureEvent(assert.AnError, nil))

			// Then
			assert.Len(t, deck.Get(), 0)
		})
	})

	t.Run("should do nothing for non-failure events", func(t *testing.T) {
		// Given
		deck := connection_deck.New()
		conn := deck.New("conn 1", "ak", "sk", "b1").Connection()

		// When
		deck.Notify(connection_deck.NewCreateSuccessEvent(deck, conn))

		// Then
		assert.Len(t, deck.Get(), 1)
	})
}

func TestDeck_NewWithOptions(t *testing.T) {
	t.Run("should create a deck with initial connections", func(t *testing.T) {
		// Given
		conn1 := connection_deck.New().New("conn 1", "ak", "sk", "b1").Connection()

		// When
		deck := connection_deck.New(connection_deck.WithConnections([]*connection_deck.Connection{conn1}))

		// Then
		assert.Len(t, deck.Get(), 1)
		assert.Equal(t, conn1, deck.Get()[0])
	})
}
