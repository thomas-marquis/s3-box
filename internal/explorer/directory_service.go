package explorer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"go.uber.org/zap"
)

type DirectoryService struct {
	logger          *zap.Logger
	repoFactory     DirectoryRepositoryFactory
	fileRepoFactory FileRepositoryFactory
	connSvc         connection.ConnectionService
}

type DirectoryRepositoryFactory func(ctx context.Context, connID uuid.UUID) (S3DirectoryRepository, error)

func NewDirectoryService(
	logger *zap.Logger,
	repoFactory DirectoryRepositoryFactory,
	fileRepoFactory FileRepositoryFactory,
	connSvc connection.ConnectionService,
) *DirectoryService {
	return &DirectoryService{
		logger:          logger,
		repoFactory:     repoFactory,
		fileRepoFactory: fileRepoFactory,
		connSvc:         connSvc,
	}
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
	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return nil, err
	}

	return repo.GetByID(ctx, id)
}

func (s *DirectoryService) getActiveRepository(ctx context.Context) (S3DirectoryRepository, error) {
	connId, err := s.connSvc.GetActiveConnectionID(ctx)
	if connId == uuid.Nil || err == ErrConnectionNoSet {
		return nil, ErrConnectionNoSet
	}
	if err != nil {
		return nil, fmt.Errorf("error whe getting directory repository: %w", err)
	}
	return s.repoFactory(ctx, connId)
}

// DeleteFile deletes a file from a directory
// It ensures the directory aggregate consistency by removing the file from the directory
// before deleting it from the repository
func (s *DirectoryService) DeleteFile(ctx context.Context, dir *S3Directory, fileID S3FileID) error {
	// First verify that the file belongs to the directory
	if !dir.HasFile(fileID) {
		return fmt.Errorf("file %s does not belong to directory %s", fileID, dir.ID)
	}

	// Get the file before removing it from the directory to be able to restore it if needed
	var fileToDelete *S3File
	for _, f := range dir.Files {
		if f.ID == fileID {
			fileToDelete = f
			break
		}
	}

	repo, err := s.getActiveRepository(ctx)
	if err != nil {
		return err
	}

	// Remove the file from the directory aggregate
	if err := dir.DeleteFile(fileID); err != nil {
		return fmt.Errorf("error removing file from directory: %w", err)
	}

	// Save the updated directory to maintain consistency
	if err := repo.Save(ctx, dir); err != nil {
		// Restore the file in the directory if saving failed
		dir.Files = append(dir.Files, fileToDelete)
		return fmt.Errorf("error saving directory: %w", err)
	}

	// Delete the file from the repository
	fileRepo, err := s.getFileRepository(ctx)
	if err != nil {
		// Restore the file in the directory if getting the file repository failed
		dir.Files = append(dir.Files, fileToDelete)
		if err := repo.Save(ctx, dir); err != nil {
			return fmt.Errorf("error restoring directory after file repository error: %w", err)
		}
		return fmt.Errorf("error getting file repository: %w", err)
	}

	if err := fileRepo.DeleteFile(ctx, fileID); err != nil {
		// Restore the file in the directory if deleting from repository failed
		dir.Files = append(dir.Files, fileToDelete)
		if err := repo.Save(ctx, dir); err != nil {
			return fmt.Errorf("error restoring directory after file deletion error: %w", err)
		}
		return fmt.Errorf("error deleting file: %w", err)
	}

	return nil
}

// getFileRepository returns the file repository for the active connection
func (s *DirectoryService) getFileRepository(ctx context.Context) (S3FileRepository, error) {
	connId, err := s.connSvc.GetActiveConnectionID(ctx)
	if connId == uuid.Nil || err == ErrConnectionNoSet {
		return nil, ErrConnectionNoSet
	}
	if err != nil {
		return nil, fmt.Errorf("error when getting file repository: %w", err)
	}
	return s.fileRepoFactory(ctx, connId)
}
