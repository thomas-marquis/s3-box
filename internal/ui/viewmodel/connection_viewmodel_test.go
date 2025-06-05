package viewmodel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

func Test_Save_ShouldSaveTheNewConnection(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockSettingsVm.EXPECT().
		CurrentTimeout().
		Return(time.Duration(10)).
		AnyTimes()

	conn1 := connection.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conn2 := connection.NewConnection(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connection.AsS3LikeConnection("localhost:9000", false),
	)

	mockConnRepo.EXPECT().
		List(gomock.AssignableToTypeOf(ctxType)).
		Return([]*connection.Connection{conn1, conn2}, nil).
		Times(1)

	newConn := connection.NewConnection(
		"connection 3",
		"POIUY",
		"98765",
		"YourBucket",
		connection.AsAWSConnection("eu-west-2"),
	)

	mockConnRepo.EXPECT().
		Save(gomock.AssignableToTypeOf(ctxType), gomock.AssignableToTypeOf(newConn)).
		Return(nil).
		Times(1)

	// When
	vm := viewmodel.NewConnectionViewModel(mockConnRepo, mockSettingsVm)
	err := vm.Save(*newConn)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, 3, vm.Connections().Length(), "Expected 3 connections in the list after saving a new connection")
	obj, _ := vm.Connections().GetValue(2)
	assert.Equal(t, *newConn, *obj.(*connection.Connection), "Expected the new connection to be the last one in the list")
}

func Test_Save_ShouldUpdateExistingConnection(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockSettingsVm.EXPECT().
		CurrentTimeout().
		Return(time.Duration(10)).
		AnyTimes()

	conn1 := connection.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conn1Updated := connection.NewConnection(
		"connection 1 updated",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conn1Updated.ID = conn1.ID // Ensure the ID remains the same for update

	conn2 := connection.NewConnection(
		"connection 2",
		"QWERTY",
		"5678",
		"OurBucket",
		connection.AsS3LikeConnection("localhost:9000", false),
	)

	mockConnRepo.EXPECT().
		List(gomock.AssignableToTypeOf(ctxType)).
		Return([]*connection.Connection{conn1, conn2}, nil).
		Times(1)

	mockConnRepo.EXPECT().
		Save(gomock.AssignableToTypeOf(ctxType), EqDeref(*conn1Updated)).
		Return(nil).
		Times(1)

	// Before updateing conn1
	vm := viewmodel.NewConnectionViewModel(mockConnRepo, mockSettingsVm)

	// When
	err := vm.Save(*conn1Updated)

	// Then after updating conn1
	assert.NoError(t, err)
	assert.Equal(t, 2, vm.Connections().Length(), "Expected 3 connections in the list after saving a new connection")
	obj, _ := vm.Connections().GetValue(0)
	assert.Equal(t, *conn1Updated, *obj.(*connection.Connection), "Expected the updated connection replacing the old one in the list")
}
