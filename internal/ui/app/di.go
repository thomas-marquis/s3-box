package app

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"go.uber.org/zap"
)

// func BuildS3DirectoryRepositoryFactory(conn *connections.Connection, log *zap.Logger, connRepository connections.Repository) explorer.DirectoryRepositoryFactory {
// 	repoById := make(map[uuid.UUID]explorer.S3DirectoryRepository)
//
// 	return func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
// 		if repo, ok := repoById[connID]; ok {
// 			return repo, nil
// 		}
//
// 		conn, err := connRepository.GetByID(ctx, connID)
// 		if err != nil {
// 			return nil, fmt.Errorf("error getting connection: %w", err)
// 		}
// 		repo, err := infrastructure.NewS3DirectoryRepositoryImpl(log, conn)
// 		if err != nil {
// 			return nil, fmt.Errorf("error creating directory repository: %w", err)
// 		}
// 		repoById[connID] = repo
// 		return repo, nil
// 	}
// }

// func BuildS3FileRepositoryFactory(conn *connections.Connection, log *zap.Logger, connRepository connections.Repository) explorer.FileRepositoryFactory {
// 	repoById := make(map[uuid.UUID]explorer.S3FileRepository)
//
// 	return func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
// 		if repo, ok := repoById[connID]; ok {
// 			return repo, nil
// 		}
//
// 		conn, err := connRepository.GetByID(ctx, connID)
// 		if err != nil {
// 			return nil, fmt.Errorf("error getting connection: %w", err)
// 		}
// 		repo, err := infrastructure.NewS3FileRepository(log, conn)
// 		if err != nil {
// 			return nil, fmt.Errorf("error creating file repository: %w", err)
// 		}
// 		repoById[connID] = repo
// 		return repo, nil
// 	}
// }

func BuildAppContext(
	connectionRepository connection_deck.Repository,
	settingsRepository settings.Repository,
	logger *zap.Logger,
	lastSelectedConn *connection_deck.Connection,
	window fyne.Window,
	initialRoute navigation.Route,
	views map[navigation.Route]func(appcontext.AppContext) (*fyne.Container, error),
	fyneSettings fyne.Settings,
) appcontext.AppContext {
	connSvc := connection_deck.NewConnectionService(connectionRepository)
	dirFactory := BuildS3DirectoryRepositoryFactory(lastSelectedConn, logger, connectionRepository)
	fileFactory := BuildS3FileRepositoryFactory(lastSelectedConn, logger, connectionRepository)
	dirSvc := explorer.NewDirectoryService(
		logger,
		dirFactory,
		fileFactory,
		connSvc,
	)
	fileSvc := explorer.NewFileService(
		logger,
		fileFactory,
		connSvc,
	)

	settingsVm := viewmodel.NewSettingsViewModel(settingsRepository, fyneSettings)
	connVm := viewmodel.NewConnectionViewModel(connectionRepository, settingsVm)
	explorerVm := viewmodel.NewExplorerViewModel(dirSvc, connectionRepository, fileSvc, settingsVm)

	return appcontext.New(window, explorerVm, connVm, settingsVm, initialRoute, views, logger, fyneSettings)
}
