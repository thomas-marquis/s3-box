package viewmodel_test

import "testing"

func Test_Save_ShouldSaveTheNewConnection(t *testing.T) {
	// Given
	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()
	// mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	// mockConnSvc := mocks_connection.NewMockConnectionService(ctrl)
	// mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)
	// vm := viewmodel.NewConnectionViewModel(mockConnRepo, mockConnSvc, mockSettingsVm)
	// newConn := connection.Connection{
	//     Name: "New Connection",
	//     Type: connection.TypeS3,
	// }
	// mockConnRepo.EXPECT().
	//     Save(gomock.Any(), gomock.Eq(newConn)).
	//     Return(nil).
	//     Times(1)
	// // When
	// err := vm.SaveConnection(newConn)
	// // Then
	// assert.NoError(t, err)
}
