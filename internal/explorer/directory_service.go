package explorer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DirectoryService struct {
	logger     *zap.Logger
	connID     uuid.UUID
	repoFactory DirectoryRepositoryFactory
}

type DirectoryRepositoryFactory func(ctx context.Context, connID uuid.UUID) (*S3DirectoryRepository, error)

func NewDirectoryService(logger *zap.Logger, repoFactory DirectoryRepositoryFactory) *DirectoryService {
	return &DirectoryService{
		logger:      logger,
		repoFactory: repoFactory,
		connID:      uuid.Nil,
	}
}

func (s *DirectoryService) SetActiveConnection(connID uuid.UUID) {
	s.connID = connID
}

func (s *DirectoryService) GetRootDirectory(ctx context.Context) (*S3Directory, error) {
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return nil, err
	}

	dir, err := repo.GetByID(ctx, RootDirID)
	if err == ErrObjectNotFound {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("impossible to get root directory: %s", err)
	}

	return dir, nil
}

func (s *DirectoryService) GetDirectoryByID(ctx context.Context, id S3DirectoryID) (*S3Directory, error) {
	if s.connID == uuid.Nil {
		return nil, ErrConnectionNoSet
	}

	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetDirectoryByID: %w", err)
	}

	return repo.GetByID(ctx, id)
}

func (s *DirectoryService) getActiveRepository(ctx context.Context) (S3DirectoryRepository, error) {
	if s.connID == uuid.Nil {
		return nil, ErrConnectionNoSet
	}
	dirRepo, err := s.repoFactory(ctx, s.connID)
	return dirRepo, err
}

// LoadDirectory loads the content of a directory from the repository
// func (s *DirectoryService) LoadDirectory(ctx context.Context, dir *S3Directory) error {
// 	repo, err := s.getActiveRepository()
// 	if err != nil {
// 		return err
// 	}

// 	// Get the directory from the repository to ensure we have the latest content
// 	latestDir, err := repo.GetByID(ctx, dir.ID)
// 	if err != nil {
// 		return fmt.Errorf("impossible to load directory: %s", err)
// 	}

// 	// Update the directory with the latest content
// 	dir.SubDirectories = latestDir.SubDirectories
// 	dir.Files = latestDir.Files
// 	dir.SubDirectoriesIDs = latestDir.SubDirectoriesIDs

// 	return nil
// }

// TODO: add a method to create a new directory (and save it) and handle the case the name contains "/"s