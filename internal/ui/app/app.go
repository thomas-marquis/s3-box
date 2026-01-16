package app

import (
	"fyne.io/fyne/v2"
	fyne_app "fyne.io/fyne/v2/app"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views"
	"go.uber.org/zap"
	"sync"
)

const (
	appId = "fr.scalde.s3box"
)

type Go2S3App struct {
	appCtx    appcontext.AppContext
	views     map[navigation.Route]appcontext.View
	initRoute navigation.Route
}

func New(logger *zap.Logger, initRoute navigation.Route) (*Go2S3App, error) {
	appViews := make(map[navigation.Route]appcontext.View)
	appViews[navigation.ExplorerRoute] = views.GetFileExplorerView
	appViews[navigation.ConnectionRoute] = views.GetConnectionView
	appViews[navigation.SettingsRoute] = views.GetSettingsView
	appViews[navigation.NotificationsRoute] = views.GetNotificationView

	sugarLog := logger.Sugar()
	a := fyne_app.NewWithID(appId)
	w := a.NewWindow("S3 Box")

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
	w.SetMainMenu(getMainMenu(appCtx))

	return &Go2S3App{
		views:     appViews,
		initRoute: initRoute,
		appCtx:    appCtx,
	}, nil
}

func (a *Go2S3App) Start() error {
	a.appCtx.Window().Resize(fyne.NewSize(1200, 900))
	err := a.appCtx.Navigate(a.initRoute)
	if err != nil {
		return err
	}
	a.appCtx.Window().ShowAndRun()
	return nil
}

func getMainMenu(appCtx appcontext.AppContext) *fyne.MainMenu {
	settingsMenu := fyne.NewMenu("Settings",
		fyne.NewMenuItem("Manage connections", func() {
			appCtx.Navigate(navigation.ConnectionRoute)
		}),
		fyne.NewMenuItem("Manage settings", func() {
			appCtx.Navigate(navigation.SettingsRoute)
		}),
		fyne.NewMenuItem("View notifications", func() {
			appCtx.Navigate(navigation.NotificationsRoute)
		}),
	)
	fileMenu := fyne.NewMenu("AttachedFile",
		fyne.NewMenuItem("Explorer view", func() {
			appCtx.Navigate(navigation.ExplorerRoute)
		}),
	)
	return fyne.NewMainMenu(fileMenu, settingsMenu)
}
