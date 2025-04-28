package viewmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	"go.uber.org/mock/gomock"
)

func TestConnectionViewModel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connRepo := mocks_connection.NewMockRepository(ctrl)
	connSvc := mocks_connection.NewMockConnectionService(ctrl)
	settingsVm := NewSettingsViewModel(nil)
	vm := NewConnectionViewModel(connRepo, connSvc, settingsVm)

	assert.NotNil(t, vm)
} 