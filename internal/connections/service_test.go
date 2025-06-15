package connections_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connections"
	"github.com/thomas-marquis/s3-box/internal/tests"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connections"
	"go.uber.org/mock/gomock"
)

func Test_Select_ShouldSelectConnectionIfExistsWhenNonAlreadySelected(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := mocks_connection.NewMockRepository(ctrl)

	conn1 := connections.New(
		"Connection 1",
		"AZERTY",
		"12345",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
	)
	conn1Selected := connections.New(
		"Connection 1",
		"AZERTY",
		"12345",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
		connections.WithID(conn1.ID()),
		connections.WithSelected(true),
	)

	conn2 := connections.New(
		"Connection 2",
		"POIUYT",
		"09876",
		"MyBucket2",
		connections.AsAWSConnection("eu-west-2"),
	)

	mockConnRepo.EXPECT().
		List(gomock.AssignableToTypeOf(tests.ContextType)).
		Return([]*connections.Connection{conn1, conn2}, nil).
		Times(1)
	mockConnRepo.EXPECT().
		Save(gomock.AssignableToTypeOf(tests.ContextType), tests.EqDeref(*conn1Selected)).
		Return(nil).
		Times(1)

	service := connections.NewConnectionService(mockConnRepo)

	// When
	ctx := context.Background()
	err := service.Select(ctx, conn1.ID())

	// Then
	assert.NoError(t, err)
}

func Test_Select_ShouldSelectConnectionIfExistsWhenOneAlreadySelected(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := mocks_connection.NewMockRepository(ctrl)

	conn1 := connections.New(
		"Connection 1",
		"AZERTY",
		"12345",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
	)
	conn1Selected := connections.New(
		"Connection 1",
		"AZERTY",
		"12345",
		"MyBucket",
		connections.AsAWSConnection("eu-west-1"),
		connections.WithID(conn1.ID()),
		connections.WithSelected(true),
	)

	conn2 := connections.New(
		"Connection 2",
		"POIUYT",
		"09876",
		"MyBucket2",
		connections.AsAWSConnection("eu-west-2"),
	)
	conn2.Select()
	conn2NotSelected := connections.New(
		"Connection 2",
		"POIUYT",
		"09876",
		"MyBucket2",
		connections.AsAWSConnection("eu-west-2"),
		connections.WithID(conn2.ID()),
		connections.WithSelected(false),
	)

	mockConnRepo.EXPECT().
		List(gomock.AssignableToTypeOf(tests.ContextType)).
		Return([]*connections.Connection{conn1, conn2}, nil).
		Times(1)
	mockConnRepo.EXPECT().
		Save(gomock.AssignableToTypeOf(tests.ContextType), tests.EqDeref(*conn1Selected)).
		Return(nil).
		Times(1)

	mockConnRepo.EXPECT().
		Save(gomock.AssignableToTypeOf(tests.ContextType), tests.EqDeref(*conn2NotSelected)).
		Return(nil).
		Times(1)

	service := connections.NewConnectionService(mockConnRepo)

	// When
	ctx := context.Background()
	err := service.Select(ctx, conn1.ID())

	// Then
	assert.NoError(t, err)
}

func Test_Select_ShouldReturnErrorWhenFailedToListConenctions(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := mocks_connection.NewMockRepository(ctrl)

	expectedErr := errors.New("CKC")

	mockConnRepo.EXPECT().
		List(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(nil, expectedErr).
		Times(1)
	mockConnRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Times(0)

	service := connections.NewConnectionService(mockConnRepo)

	// When
	ctx := context.Background()
	err := service.Select(ctx, uuid.New())

	// Then
	assert.Equal(t, expectedErr, err)

}
