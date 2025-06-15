package connections_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connections"
)

func Test_Delete_ShouldDeleteConnection(t *testing.T) {
	// Given
	conn1 := connections.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
	)
	conn2 := connections.New(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connections.AsS3LikeConnection("localhost:9000", false),
	)

	conns := connections.NewSet(connections.WithConnections(
		[]*connections.Connection{conn1, conn2},
	))

	// When
	res := conns.Delete(conn1.ID())

	// Then
	assert.NoError(t, res)
	assert.Len(t, conns.Connections(), 1)
}

func Test_Delete_ShouldReturnErrorIfConnectionNotFound(t *testing.T) {
	// Given
	conn1 := connections.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
	)
	conns := connections.NewSet(connections.WithConnections(
		[]*connections.Connection{conn1},
	))

	// When
	res := conns.Delete(uuid.New())

	// Then
	assert.Error(t, res)
	assert.Equal(t, connections.ErrConnectionNotFound, res)
	assert.Len(t, conns.Connections(), 1)
}

func Test_Select_ShouldSelectConnection(t *testing.T) {
	// Given
	conn1 := connections.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
		connections.WithSelected(true),
	)
	conn2 := connections.New(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connections.AsS3LikeConnection("localhost:9000", false),
		connections.WithSelected(false),
	)

	conns := connections.NewSet(connections.WithConnections(
		[]*connections.Connection{conn1, conn2},
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
	conn1 := connections.New(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
		connections.WithSelected(true),
	)

	conns := connections.NewSet(connections.WithConnections(
		[]*connections.Connection{conn1},
	))

	// When
	err := conns.Select(uuid.New())

	// Then
	assert.Error(t, err)
	assert.Equal(t, connections.ErrConnectionNotFound, err)
	assert.True(t, conn1.Selected(), "Expected conn1 to remain selected")
}
