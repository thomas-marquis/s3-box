package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	"go.uber.org/zap"
)

func BuildS3DirectoryRepositoryFactory(conn *connection.Connection, log *zap.Logger, connRepository connection.Repository) explorer.DirectoryRepositoryFactory {
	repoById := make(map[uuid.UUID]*explorer.S3DirectoryRepository)

	return func(ctx context.Context, connID uuid.UUID) (*explorer.S3DirectoryRepository, error) {
		if repo, ok := repoById[connID]; ok {
			return repo, nil
		}

		conn, err := connRepository.GetByID(ctx, connID)
		if err != nil {
			return nil, fmt.Errorf("error getting connection: %w", err)
		}
		repo, err := infrastructure.NewS3DirectoryRepositoryImpl(log, conn)
		if err != nil {
			return nil, fmt.Errorf("error creating directory repository: %w", err)
		}
		repoById[connID] = repo
		return repo, nil
	}
}