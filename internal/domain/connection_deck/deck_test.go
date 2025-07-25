package connection_deck_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

func Test_RemoveAConnection_ShouldRemoveTheGivenConnection(t *testing.T) {
	// Given
	conns := connection_deck.New()
	conn1 := conns.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection_deck.AsAWS("eu-west-1"),
	)
	conns.New(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connection_deck.AsS3Like("localhost:9000", false),
	)

	// When
	res := conns.RemoveAConnection(conn1.ID())

	// Then
	assert.NoError(t, res)
	assert.Len(t, conns.Get(), 1)
}

func Test_RemoveAConnection_ShouldReturnErrorIfConnectionNotFound(t *testing.T) {
	// Given
	conns := connection_deck.New()
	conns.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection_deck.AsAWS("eu-west-1"),
	)

	// When
	res := conns.RemoveAConnection(connection_deck.NewConnectionID())

	// Then
	assert.Error(t, res)
	assert.Equal(t, connection_deck.ErrNotFound, res)
	assert.Len(t, conns.Get(), 1)
}

func Test_Select_ShouldSelectConnection(t *testing.T) {
	// Given
	conns := connection_deck.New()
	conns.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection_deck.AsAWS("eu-west-1"),
	)
	conn2 := conns.New(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connection_deck.AsS3Like("localhost:9000", false),
	)

	// When
	err := conns.Select(conn2.ID())

	// Then
	assert.NoError(t, err)
	assert.Equal(t, conn2, conns.SelectedConnection())
}

func Test_Select_ShouldReturnErrorWhenConnectionNotFound(t *testing.T) {
	// Given
	conns := connection_deck.New()
	conns.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection_deck.AsAWS("eu-west-1"),
	)

	// When
	err := conns.Select(connection_deck.NewConnectionID())

	// Then
	assert.Error(t, err)
	assert.Equal(t, connection_deck.ErrNotFound, err)
	assert.Nil(t, conns.SelectedConnection())
}
