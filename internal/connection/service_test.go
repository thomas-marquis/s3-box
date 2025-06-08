package connection_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/tests"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	"go.uber.org/mock/gomock"
)

func Test_Select_ShouldSelectConnectionIfExistsWhenNonAlreadySelected(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := mocks_connection.NewMockRepository(ctrl)

	conn1 := connection.NewConnection(
		"Connection 1",
		"AZERTY",
		"12345",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conn2 := connection.NewConnection(
		"Connection 2",
		"POIUYT",
		"09876",
		"MyBucket2",
		connection.AsAWSConnection("eu-west-2"),
	)

	mockConnRepo.EXPECT().
		List(gomock.AssignableToTypeOf(tests.ContextType)).
		Return([]*connection.Connection{conn1, conn2}, nil).
		Times(1)
	mockConnRepo.EXPECT().
		SetSelected(tests.ContextType, conn1.ID).
		Return(nil).
		Times(1)

	service := connection.NewConnectionService(mockConnRepo)

	// When
	ctx := context.Background()
	err := service.Select(ctx, conn1.ID)

	// Then
	assert.NoError(t, err)
}
