package app

import (
	"sync"

	"fyne.io/fyne/v2"
	fyne_app "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views"
	"go.uber.org/zap"
)

const (
	appId   = "fr.scalde.s3box"
	appName = "S3 Box"
)

type Go2S3App struct {
	appCtx        appcontext.AppContext
	initRoute     navigation.Route
	windowContent fyne.CanvasObject
}

func New(logger *zap.Logger, initRoute navigation.Route) (*Go2S3App, error) {
	appViews := map[navigation.Route]appcontext.Menu{
		navigation.ExplorerRoute: {
			Label:       "File explorer",
			IconFactory: theme.HomeIcon,
			View:        views.GetFileExplorerView,
			Route:       navigation.ExplorerRoute,
			Index:       0,
		},
		navigation.ConnectionRoute: {
			Label:       "Connections",
			IconFactory: theme.StorageIcon,
			View:        views.GetConnectionView,
			Route:       navigation.ConnectionRoute,
			Index:       1,
		},
		navigation.SettingsRoute: {
			Label:       "Settings",
			IconFactory: theme.SettingsIcon,
			View:        views.GetSettingsView,
			Route:       navigation.SettingsRoute,
			Index:       2,
		},
		navigation.NotificationsRoute: {
			Label:       "Notifications",
			IconFactory: theme.InfoIcon,
			View:        views.GetNotificationView,
			Route:       navigation.NotificationsRoute,
			Index:       3,
		},
	}

	sugarLog := logger.Sugar()
	a := fyne_app.NewWithID(appId)
	a.Settings().SetTheme(apptheme.Get(a.Settings().ThemeVariant()))
	w := a.NewWindow(appName)

	terminated := make(chan struct{})
	eventBus := newEventBusImpl(terminated)

	notifier := infrastructure.NewNotificationPublisher()

	fyneSettings := a.Settings()

	connectionsRepository := infrastructure.NewFyneConnectionsRepository(a.Preferences(), eventBus)

	settingsRepository := infrastructure.NewSettingsRepository(a.Preferences())

	directoryRepository, err := infrastructure.NewS3DirectoryRepository(
		connectionsRepository,
		eventBus,
		notifier,
	)
	if err != nil {
		sugarLog.Error("Error creating directory repository", err)
		return nil, err
	}

	settingsViewModel := viewmodel.NewSettingsViewModel(settingsRepository, fyneSettings, notifier)
	connectionViewModel := viewmodel.NewConnectionViewModel(
		connectionsRepository,
		settingsViewModel,
		notifier,
		eventBus,
	)
	explorerViewModel := viewmodel.NewExplorerViewModel(
		directoryRepository,
		settingsViewModel,
		notifier,
		connectionViewModel.Deck().SelectedConnection(),
		eventBus,
	)

	notificationsViewModel := viewmodel.NewNotificationViewModel(notifier, terminated)

	appCtx := appcontext.New(
		appName,
		w,
		explorerViewModel,
		connectionViewModel,
		settingsViewModel,
		notificationsViewModel,
		initRoute,
		appViews,
		logger,
		fyneSettings,
		eventBus,
	)

	var one sync.Once
	w.SetOnClosed(func() {
		one.Do(func() {
			close(terminated)
		})
	})

	return &Go2S3App{
		initRoute: initRoute,
		appCtx:    appCtx,
	}, nil
}

func (a *Go2S3App) Start() error {
	a.appCtx.Window().Resize(fyne.NewSize(1200, 900))
	a.appCtx.Window().SetContent(a.appCtx.AppContent())
	_, err := a.appCtx.Navigate(a.initRoute)
	if err != nil {
		return err
	}
	a.appCtx.Window().ShowAndRun() // blocking
	return nil
}
