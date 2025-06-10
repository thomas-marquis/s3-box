package viewmodel_test

import (
	"fmt"
	"testing"
	"time"

	_ "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/tests"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_RefreshDir_ShouldRefreshDirectoryContent(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup viewmodel
	mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	mockDirSvc := mocks_explorer.NewMockDirectoryService(ctrl)
	mockFileSvc := mocks_explorer.NewMockFileService(ctrl)
	mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockSettingsVm.EXPECT().
		CurrentTimeout().
		Return(time.Duration(10)).
		AnyTimes()

	// setup fake directory structure
	fakeRootDir, _ := explorer.NewS3Directory(explorer.RootDirName, explorer.NilParentID)
	fakeFile, _ := explorer.NewS3File("config.txt", fakeRootDir)
	fakeRootDir.AddSubDirectory("subdir")
	fakeRootDir.AddFile(fakeFile)

	mockDirSvc.EXPECT().
		GetRootDirectory(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(fakeRootDir, nil).
		Times(1)
	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(explorer.RootDirID)).
		Return(fakeRootDir, nil).
		Times(1)

	fakeSubdir, _ := explorer.NewS3Directory("subdir", fakeRootDir.ID)
	fakeSubFile, _ := explorer.NewS3File("subfile.txt", fakeSubdir)
	fakeSubdir.AddFile(fakeSubFile)
	fakeSubdir.AddSubDirectory("demo")

	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(explorer.S3DirectoryID("/subdir/"))).
		Return(fakeSubdir, nil).
		Times(1)

	// setup fake connection
	fakeConn := connection.New(
		"my connection",
		"12345",
		"AZERTY",
		"MyBucket",
		connection.AsAWSConnection("eu-west-3"),
	)

	mockConnRepo.EXPECT().
		GetSelected(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(fakeConn, nil).
		AnyTimes()

	expectedTree := tests.NewUntypedTreeBuilder().
		WithLoadedRootNode("Bucket: MyBucket").
		WithLoadedFileNode("/config.txt", "/", "config.txt").
		WithLoadedDirNode("/subdir/", "/", "subdir").
		WithLoadedFileNode("/subdir/subfile.txt", "/subdir/", "subfile.txt").
		WithDirNode("/subdir/demo/", "/subdir/", "demo").
		Build()

	// When
	vm := viewmodel.NewExplorerViewModel(mockDirSvc, mockConnRepo, mockFileSvc, mockSettingsVm)
	// vm.RefreshDir(explorer.RootDirID)
	err := vm.RefreshDir(explorer.S3DirectoryID("/subdir/"))

	// Then
	assert.NoError(t, err)
	ok, report := tests.AreTreesEqual(vm.Tree(), expectedTree)
	fmt.Println(report)
	assert.True(t, ok, "The tree structure should be equal to the expected one")
}

func Test_CreateEmptyDirectory_ShouldCreateNewDirAtRootAndAddItToTree(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup viewmodel
	mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	mockDirSvc := mocks_explorer.NewMockDirectoryService(ctrl)
	mockFileSvc := mocks_explorer.NewMockFileService(ctrl)
	mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockSettingsVm.EXPECT().
		CurrentTimeout().
		Return(time.Duration(10)).
		AnyTimes()

	// setup fake directory structure
	fakeRootDir, _ := explorer.NewS3Directory(explorer.RootDirName, explorer.NilParentID)
	fakeFile, _ := explorer.NewS3File("config.txt", fakeRootDir)
	fakeRootDir.AddSubDirectory("subdir")
	fakeRootDir.AddFile(fakeFile)

	mockDirSvc.EXPECT().
		GetRootDirectory(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(fakeRootDir, nil).
		Times(1)
	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(explorer.RootDirID)).
		Return(fakeRootDir, nil).
		Times(1)

	// setup fake connection
	fakeConn := connection.New(
		"my connection",
		"12345",
		"AZERTY",
		"MyBucket",
		connection.AsAWSConnection("eu-west-3"),
	)

	mockConnRepo.EXPECT().
		GetSelected(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(fakeConn, nil).
		AnyTimes()

	fakeNewDir, _ := explorer.NewS3Directory("newDir", fakeRootDir.ID)

	mockDirSvc.EXPECT().
		CreateSubDirectory(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(fakeRootDir), gomock.Eq("newDir")).
		Return(fakeNewDir, nil).
		Times(1)

	expectedTree := tests.NewUntypedTreeBuilder().
		WithLoadedRootNode("Bucket: MyBucket").
		WithLoadedFileNode("/config.txt", "/", "config.txt").
		WithDirNode("/subdir/", "/", "subdir").
		WithDirNode("/newDir/", "/", "newDir").
		Build()

	// When
	vm := viewmodel.NewExplorerViewModel(mockDirSvc, mockConnRepo, mockFileSvc, mockSettingsVm)
	res, err := vm.CreateEmptyDirectory(fakeRootDir, "newDir")

	// Then
	assert.NoError(t, err, "Unexpected error when creating new directory")
	assert.Equal(t, fakeNewDir, res, "The created directory should be the same as the returned one")
	treesEqual, report := tests.AreTreesEqual(vm.Tree(), expectedTree)
	fmt.Println(report)
	assert.True(t, treesEqual, "The tree structure should be equal to the expected one")
}

func Test_CreateEmptyDirectory_ShouldCreateNewDirUnderOtherDirAndAddItToTree(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup viewmodel
	mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	mockDirSvc := mocks_explorer.NewMockDirectoryService(ctrl)
	mockFileSvc := mocks_explorer.NewMockFileService(ctrl)
	mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockSettingsVm.EXPECT().
		CurrentTimeout().
		Return(time.Duration(10)).
		AnyTimes()

	// setup fake directory structure
	fakeRootDir, _ := explorer.NewS3Directory(explorer.RootDirName, explorer.NilParentID)
	fakeFile, _ := explorer.NewS3File("config.txt", fakeRootDir)
	fakeRootDir.AddSubDirectory("subdir")
	fakeRootDir.AddFile(fakeFile)

	mockDirSvc.EXPECT().
		GetRootDirectory(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(fakeRootDir, nil).
		Times(1)
	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(explorer.RootDirID)).
		Return(fakeRootDir, nil).
		Times(1)

	fakeSubDir, _ := explorer.NewS3Directory("subdir", explorer.RootDirID)

	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(explorer.S3DirectoryID("/subdir/"))).
		Return(fakeSubDir, nil).
		Times(1)

	// setup fake connection
	fakeConn := connection.New(
		"my connection",
		"12345",
		"AZERTY",
		"MyBucket",
		connection.AsAWSConnection("eu-west-3"),
	)

	mockConnRepo.EXPECT().
		GetSelected(gomock.AssignableToTypeOf(tests.ContextType)).
		Return(fakeConn, nil).
		AnyTimes()

	fakeNewDir, _ := explorer.NewS3Directory("newDir", fakeSubDir.ID)

	mockDirSvc.EXPECT().
		CreateSubDirectory(gomock.AssignableToTypeOf(tests.ContextType), gomock.Eq(fakeSubDir), gomock.Eq("newDir")).
		Return(fakeNewDir, nil).
		Times(1)

	expectedTree := tests.NewUntypedTreeBuilder().
		WithLoadedRootNode("Bucket: MyBucket").
		WithLoadedFileNode("/config.txt", "/", "config.txt").
		WithLoadedDirNode("/subdir/", "/", "subdir").
		WithDirNode("/subdir/newDir/", "/subdir/", "newDir").
		Build()

		// When
	vm := viewmodel.NewExplorerViewModel(mockDirSvc, mockConnRepo, mockFileSvc, mockSettingsVm)
	// Navigate to subdir
	errNav := vm.OpenDirectory(fakeSubDir.ID)
	// Create a new directory under subdir
	res, err := vm.CreateEmptyDirectory(fakeSubDir, "newDir")

	// Then
	assert.NoError(t, errNav, "Should not return an error when navigating to subdir")
	assert.NoError(t, err, "Unexpected error when creating new directory")
	assert.Equal(t, fakeNewDir, res, "The created directory should be the same as the returned one")
	treesEqual, report := tests.AreTreesEqual(vm.Tree(), expectedTree)
	fmt.Println(report)
	assert.True(t, treesEqual, "The tree structure should be equal to the expected one")
}
