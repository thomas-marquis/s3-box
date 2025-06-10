package connection_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connection"
)

func Test_Delete_ShouldDeleteConnection(t *testing.T) {
	// Given
	conn1 := connection.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conn2 := connection.New(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connection.AsS3LikeConnection("localhost:9000", false),
	)

	conns := connection.NewSet(connection.WithConnections(
		[]*connection.Connection{conn1, conn2},
	))

	// When
	res := conns.Delete(conn1.ID())

	// Then
	assert.NoError(t, res)
	assert.Len(t, conns.Connections(), 1)
}

func Test_Delete_ShouldReturnErrorIfConnectionNotFound(t *testing.T) {
	// Given
	conn1 := connection.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conns := connection.NewSet(connection.WithConnections(
		[]*connection.Connection{conn1},
	))

	// When
	res := conns.Delete(uuid.New())

	// Then
	assert.Error(t, res)
	assert.Equal(t, connection.ErrConnectionNotFound, res)
	assert.Len(t, conns.Connections(), 1)
}

func Test_Select_ShouldSelectConnection(t *testing.T) {
	// Given
	conn1 := connection.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
		connection.WithSelected(true),
	)
	conn2 := connection.New(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connection.AsS3LikeConnection("localhost:9000", false),
		connection.WithSelected(false),
	)

	conns := connection.NewSet(connection.WithConnections(
		[]*connection.Connection{conn1, conn2},
	))

	// When
	err := conns.Select(conn2.ID())

	// Then
	assert.NoError(t, err)
	assert.True(t, conn2.Selected())
	assert.False(t, conn1.Selected())
	assert.True(t, conns.Selected().Is(conn2), "Expected the selected connection to be conn2")
}

func Test_Select_ShouldReturnErrorWhenConnectionNotFound(t *testing.T) {
	// Given
	conn1 := connection.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
		connection.WithSelected(true),
	)

	conns := connection.NewSet(connection.WithConnections(
		[]*connection.Connection{conn1},
	))

	// When
	err := conns.Select(uuid.New())

	// Then
	assert.Error(t, err)
	assert.Equal(t, connection.ErrConnectionNotFound, err)
	assert.True(t, conn1.Selected(), "Expected conn1 to remain selected")
}
