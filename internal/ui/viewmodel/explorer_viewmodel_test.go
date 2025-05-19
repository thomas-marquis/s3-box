package viewmodel_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func areTreesEqual(actual binding.UntypedTree, expected binding.UntypedTree) bool {
	compare := func(a, b binding.UntypedTree, aLabel, bLabel string) bool {
		res := true
		_, aTreeContent, _ := a.Get()
		for i := range aTreeContent {
			val, _ := a.GetValue(i)
			aNode, aOk := val.(*viewmodel.TreeNode)
			if !aOk {
				fmt.Printf("Error casting %s node (ID=%s; Value=%v) as a pointer of viewmodel.TreeNode\n", aLabel, i, val)
				res = false
				continue
			}
			val, err := b.GetValue(i)
			bNode, bOk := val.(*viewmodel.TreeNode)
			if !bOk {
				fmt.Printf("Error casting %s node (ID=%s; Value=%v) as a pointer of viewmodel.TreeNode\n", bLabel, i, val)
				res = false
			} else if err != nil {
				fmt.Printf("Error getting %s node: %v\n", bLabel, err)
				res = false
			} else if *aNode != *bNode {
				fmt.Printf("Node with id %s mismatch: %s: %v, %s: %v\n", i, aLabel, aNode, bLabel, bNode)
				res = false
			}
		}
		return res
	}

	return compare(actual, expected, "actual", "expected") && compare(expected, actual, "expected", "actual")
}

func Test_areTreesEqual_ShouldReturnTrueWhenTreesAreEqual(t *testing.T) {
	// Given a first tree
	tree1 := binding.NewUntypedTree()
	tree1.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree1.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree1.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// Given a second tree
	tree2 := binding.NewUntypedTree()
	tree2.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree2.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree2.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// When
	result := areTreesEqual(tree1, tree2)

	// Then
	assert.True(t, result, "The trees should be equal")
}

func Test_assertTreeContent_ShouldReturnFalseWhenATreeNotContainsPointers(t *testing.T) {
	// Given a first tree
	tree1 := binding.NewUntypedTree()
	tree1.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree1.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree1.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// Given a second tree
	tree2 := binding.NewUntypedTree()
	tree2.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree2.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree2.Append("/", "/file.txt", *viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// When
	result := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
}

func Test_areTreesEqual_SHouldReturnFalseWhenTreesAreNotEqual(t *testing.T) {
	// Given a first tree
	tree1 := binding.NewUntypedTree()
	tree1.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree1.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree1.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// Given a second tree with different content
	tree2 := binding.NewUntypedTree()
	tree2.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree2.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree2.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "differentFile.txt", viewmodel.TreeNodeTypeFile))

	// When
	result := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
}

func Test_areTreesEqual_ShouldReturnFalseWhenLessNodesInSecondTree(t *testing.T) {
	// Given a first tree
	tree1 := binding.NewUntypedTree()
	tree1.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree1.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree1.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// Given a second tree with less nodes
	tree2 := binding.NewUntypedTree()
	tree2.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree2.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))

	// When
	result := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
}

func Test_areTreesEqual_ShouldReturnFalseWhenLessNodesInFirstTree(t *testing.T) {
	// Given a first tree with less nodes
	tree1 := binding.NewUntypedTree()
	tree1.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree1.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))

	// Given a second tree
	tree2 := binding.NewUntypedTree()
	tree2.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree2.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree2.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// When
	result := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
}

func Test_areTreesEqual_ShouldReturnTrueWhenSameTreesButDifferentOrder(t *testing.T) {
	// Given a first tree
	tree1 := binding.NewUntypedTree()
	tree1.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree1.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))
	tree1.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))

	// Given a second tree with different order
	tree2 := binding.NewUntypedTree()
	tree2.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	tree2.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	tree2.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// When
	result := areTreesEqual(tree1, tree2)

	// Then
	assert.True(t, result, "The trees should be equal")
}

func Test_RefreshDir_ShouldRefreshDirectoryContent(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

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
		GetRootDirectory(gomock.AssignableToTypeOf(ctxType)).
		Return(fakeRootDir, nil).
		Times(1)
	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(ctxType), gomock.Eq(explorer.RootDirID)).
		Return(fakeRootDir, nil).
		Times(1)

	fakeSubdir, _ := explorer.NewS3Directory("subdir", fakeRootDir.ID)

	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(ctxType), gomock.Eq(explorer.S3DirectoryID("/subdir"))).
		Return(fakeSubdir, nil).
		Times(1)

	// setup fake connection
	fakeConn := connection.NewConnection(
		"my connection",
		"12345",
		"AZERTY",
		"MyBucket",
		connection.AsAWSConnection("eu-west-3"),
	)

	mockConnRepo.EXPECT().
		GetSelectedConnection(gomock.AssignableToTypeOf(ctxType)).
		Return(fakeConn, nil).
		AnyTimes()

	expectedTree := binding.NewUntypedTree()
	expectedTree.Append("", "/", viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot))
	expectedTree.Append("/", "/someDir", viewmodel.NewTreeNode("/someDir", "someDir", viewmodel.TreeNodeTypeDirectory))
	expectedTree.Append("/", "/file.txt", viewmodel.NewTreeNode("/file.txt", "file.txt", viewmodel.TreeNodeTypeFile))

	// When
	vm := viewmodel.NewExplorerViewModel(mockDirSvc, mockConnRepo, mockFileSvc, mockSettingsVm)
	err := vm.RefreshDir(explorer.S3DirectoryID("/subdir"))

	// Then
	assert.NoError(t, err)
	assert.True(t, areTreesEqual(vm.Tree(), expectedTree), "The tree structure should be equal to the expected one")
}

// func Test_RefreshDir_ShouldHandleErrorFromDirectoryService(t *testing.T) {
// 	// Given
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()
//
// 	logger := zap.NewNop()
// 	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
// 	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
// 	connSvc := mocks_connection.NewMockConnectionService(ctrl)
// 	connRepo := mocks_connection.NewMockRepository(ctrl)
// 	dirSvc := explorer.NewDirectoryService(
// 		logger,
// 		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
// 			return dirRepo, nil
// 		},
// 		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
// 			return fileRepo, nil
// 		},
// 		connSvc,
// 	)
// 	settingsVm := viewmodel.NewSettingsViewModel(nil)
// 	vm := viewmodel.NewExplorerViewModel(dirSvc, connRepo, nil, settingsVm)
//
// 	dirID := explorer.S3DirectoryID("/test")
// 	connID := uuid.New()
// 	rootDir := &explorer.S3Directory{
// 		ID:   explorer.RootDirID,
// 		Name: "",
// 	}
// 	conn := &connection.Connection{
// 		BucketName: "test-bucket",
// 	}
//
// 	// Expectations
// 	connRepo.EXPECT().
// 		GetSelectedConnection(gomock.Any()).
// 		Return(conn, nil).
// 		Times(2)
// 	connSvc.EXPECT().
// 		GetActiveConnectionID(gomock.Any()).
// 		Return(connID, nil).
// 		Times(3)
// 	dirRepo.EXPECT().
// 		GetByID(gomock.Any(), explorer.RootDirID).
// 		Return(rootDir, nil).
// 		Times(2)
// 	dirRepo.EXPECT().
// 		GetByID(gomock.Any(), dirID).
// 		Return(nil, explorer.ErrConnectionNoSet).
// 		Times(1)
//
// 	// When
// 	err := vm.RefreshDir(dirID)
//
// 	// Then
// 	assert.Error(t, err)
// 	assert.Equal(t, explorer.ErrConnectionNoSet, err)
// }
//
// func Test_RefreshDir_ShouldHandleErrorFromTreeOperations(t *testing.T) {
// 	// Given
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()
//
// 	logger := zap.NewNop()
// 	dirRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
// 	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
// 	connSvc := mocks_connection.NewMockConnectionService(ctrl)
// 	connRepo := mocks_connection.NewMockRepository(ctrl)
// 	dirSvc := explorer.NewDirectoryService(
// 		logger,
// 		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
// 			return dirRepo, nil
// 		},
// 		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
// 			return fileRepo, nil
// 		},
// 		connSvc,
// 	)
// 	settingsVm := viewmodel.NewSettingsViewModel(nil)
// 	vm := viewmodel.NewExplorerViewModel(dirSvc, connRepo, nil, settingsVm)
//
// 	dirID := explorer.S3DirectoryID("/test")
// 	dir := &explorer.S3Directory{
// 		ID:   dirID,
// 		Name: "test",
// 		Files: []*explorer.S3File{
// 			{ID: "test/file.txt", Name: "file.txt"},
// 		},
// 	}
// 	connID := uuid.New()
// 	rootDir := &explorer.S3Directory{
// 		ID:   explorer.RootDirID,
// 		Name: "",
// 	}
// 	conn := &connection.Connection{
// 		BucketName: "test-bucket",
// 	}
//
// 	// Expectations
// 	connRepo.EXPECT().
// 		GetSelectedConnection(gomock.Any()).
// 		Return(conn, nil).
// 		Times(2)
// 	connSvc.EXPECT().
// 		GetActiveConnectionID(gomock.Any()).
// 		Return(connID, nil).
// 		Times(3)
// 	dirRepo.EXPECT().
// 		GetByID(gomock.Any(), explorer.RootDirID).
// 		Return(rootDir, nil).
// 		Times(2)
// 	dirRepo.EXPECT().
// 		GetByID(gomock.Any(), dirID).
// 		Return(dir, nil).
// 		Times(1)
//
// 	// When
// 	err := vm.RefreshDir(dirID)
//
// 	// Then
// 	assert.NoError(t, err) // Les erreurs d'arbre sont loggées mais ne font pas échouer l'opération
// }
