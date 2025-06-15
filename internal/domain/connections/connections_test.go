package connections_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
)

func Test_RemoveAConnection_ShouldRemoveTheGivenConnection(t *testing.T) {
	// Given
	conns := connections.New()
	conn1 := conns.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWS("eu-west-1"),
	)
	conns.NewConnection(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connections.AsS3Like("localhost:9000", false),
	)

	// When
	res := conns.RemoveAConnection(conn1.ID())

	// Then
	assert.NoError(t, res)
	assert.Len(t, conns.Get(), 1)
}

func Test_RemoveAConnection_ShouldReturnErrorIfConnectionNotFound(t *testing.T) {
	// Given
	conns := connections.New()
	conns.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWS("eu-west-1"),
	)

	// When
	res := conns.RemoveAConnection(connections.NewConnectionID())

	// Then
	assert.Error(t, res)
	assert.Equal(t, connections.ErrNotFound, res)
	assert.Len(t, conns.Get(), 1)
}

func Test_Select_ShouldSelectConnection(t *testing.T) {
	// Given
	conns := connections.New()
	conns.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWS("eu-west-1"),
	)
	conn2 := conns.NewConnection(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connections.AsS3Like("localhost:9000", false),
	)

	// When
	err := conns.Select(conn2.ID())

	// Then
	assert.NoError(t, err)
	assert.Equal(t, conn2, conns.SelectedConnection())
}

func Test_Select_ShouldReturnErrorWhenConnectionNotFound(t *testing.T) {
	// Given
	conns := connections.New()
	conns.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWS("eu-west-1"),
	)

	// When
	err := conns.Select(connections.NewConnectionID())

	// Then
	assert.Error(t, err)
	assert.Equal(t, connections.ErrNotFound, err)
	assert.Nil(t, conns.SelectedConnection())
}
