package state

import (
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
)

type ExplorerState struct {
	fileTree binding.Tree[node.Node]
}

func (s *ExplorerState) FileTree() binding.Tree[node.Node] {
	return s.fileTree
}

func (s *ExplorerState) InitFileTree(rootDir *directory.Directory, bucketName string) error {
	s.fileTree = binding.NewTree[node.Node](func(n1 node.Node, n2 node.Node) bool {
		return n1.ID() == n2.ID()
	})

	displayLabel := "Bucket: " + bucketName
	rootNode := node.NewDirectoryNode(rootDir, node.WithDisplayName(displayLabel))
	if err := s.fileTree.Append("", rootNode.ID(), rootNode); err != nil {
		return NewError("failed adding root directory to file tree", err)
	}

	return nil
}

func (s *ExplorerState) ClearChildren(dir *directory.Directory) {
	var subNodePaths []string
	for _, sd := range dir.SubDirectories() {
		subNodePaths = append(subNodePaths, sd.Path().String())
	}
	for _, f := range dir.Files() {
		subNodePaths = append(subNodePaths, f.FullPath())
	}

	for _, p := range subNodePaths {
		if err := s.fileTree.Remove(p); err != nil {
			continue
		}
	}
}

func (s *ExplorerState) CreateChildren(dir *directory.Directory) {
	files := dir.Files()
	subDirs := dir.SubDirectories()

	for _, subDir := range subDirs {
		subDirNode := node.NewDirectoryNode(subDir)
		if err := s.fileTree.Append(dir.Path().String(), subDirNode.ID(), subDirNode); err != nil {
			logger.Printf("error appending subdirectory to tree: %s", err)
			continue
		}
		s.CreateChildren(subDir)
	}

	for _, file := range files {
		fileNode := node.NewFileNode(file)
		if err := s.fileTree.Append(dir.Path().String(), fileNode.ID(), fileNode); err != nil {
			logger.Printf("error appending file to tree: %s", err)
			continue
		}
	}
}

func (s *ExplorerState) UpdateChildren(dir *directory.Directory) {
	s.ClearChildren(dir)
	s.CreateChildren(dir)
}

func (s *ExplorerState) UpdateOrPrepend(dir *directory.Directory) error {
	n := node.NewDirectoryNode(dir)
	if s.IsNodeExists(n.ID()) {
		if err := s.UpdateNode(n.ID(), n); err != nil {
			return err
		}
		return nil
	}
	if err := s.fileTree.Prepend(dir.Parent().Path().String(), n.ID(), n); err != nil {
		return NewError(fmt.Sprintf("failed adding new directory '%s' to file tree", dir.Name()), err)
	}
	return nil
}

func (s *ExplorerState) UpdateOrAppendFile(f *directory.File) error {
	nodePath := f.FullPath()
	if s.IsNodeExists(nodePath) {
		if err := s.fileTree.SetValue(nodePath, node.NewFileNode(f)); err != nil {
			return NewError(fmt.Sprintf("failed updating file '%s' in file tree", f.Name()), err)
		}
		return nil
	}
	return s.AppendFile(f)
}

func (s *ExplorerState) AppendFile(f *directory.File) error {
	newFileNode := node.NewFileNode(f)
	if err := s.fileTree.Append(f.DirectoryPath().String(), newFileNode.ID(), newFileNode); err != nil {
		return NewError(fmt.Sprintf("failed appending the file '%s' to the tree", f.Name()), err)
	}
	return nil
}

func (s *ExplorerState) PrependDirectory(dir *directory.Directory) error {
	isParentExists := s.IsNodeExists(dir.Parent().Path().String())
	if !isParentExists {
		return NewError(fmt.Sprintf("failed prepending the directory '%s' to file tree because its parents has not been found",
			dir.Path()))
	}

	n := node.NewDirectoryNode(dir)
	if err := s.fileTree.Prepend(dir.Parent().Path().String(), n.ID(), n); err != nil {
		return NewError(fmt.Sprintf("failed prepending the directory '%s' to file tree", dir.Name()), err)
	}
	return nil
}

func (s *ExplorerState) RemoveNode(nodeID string) error {
	if err := s.fileTree.Remove(nodeID); err != nil {
		return NewError(fmt.Sprintf("failed removing node '%s' from file tree", nodeID), err)
	}
	return nil
}

func (s *ExplorerState) IsNodeExists(nodeID string) bool {
	_, err := s.fileTree.GetValue(nodeID)
	return err == nil
}

func (s *ExplorerState) UpdateNode(nodeID string, newNode node.Node) error {
	if err := s.fileTree.SetValue(nodeID, newNode); err != nil {
		return NewError(fmt.Sprintf("failed updating node %s from file tree", nodeID), err)
	}
	return nil
}

func (s *ExplorerState) GetDirectoryNode(path directory.Path) (node.DirectoryNode, error) {
	n, err := s.fileTree.GetValue(path.String())
	if err != nil {
		return nil, NewError(fmt.Sprintf("directory '%s' not found in the file tree", path.String()))
	}
	dirNode, ok := n.(node.DirectoryNode)
	if !ok {
		return nil, NewError(fmt.Sprintf("call the police, node '%s' is not a directory", path.String()))
	}
	return dirNode, nil
}
