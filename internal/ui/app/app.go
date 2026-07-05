package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	fyne_app "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/settings"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
	"github.com/thomas-marquis/s3-box/internal/ui/theme/resources"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views"
	"go.uber.org/zap"
)

const (
	appId   = "fr.scalde.s3box"
	appName = "S3 Box"
)

type Go2S3App struct {
	appCtx    appcontext.AppContext
	initRoute navigation.Route
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

	a := fyne_app.NewWithID(appId)
	a.Settings().SetTheme(apptheme.GetDefaultByVariant(a.Settings().ThemeVariant()))
	a.SetIcon(resources.NewAppLogo())

	w := a.NewWindow(appName)

	ctx, cancel := context.WithCancel(context.Background())
	notifier := infrastructure.NewNotificationPublisher(notification.LevelDebug)

	eventBus := inmemory.NewBus(ctx, inmemory.WithNotifier(&notifierAdapter{notifier: notifier}))

	fyneSettings := a.Settings()

	connectionsRepository := infrastructure.NewFyneConnectionsRepository(a.Preferences(), eventBus)

	s3.NewS3EventHandler(
		connectionsRepository,
		eventBus,
		notifier,
	).Listen()

	settings.FyneSettingsHandler(eventBus, a.Preferences())

	appState := state.New()

	notificationsViewModel := viewmodel.NewNotificationViewModel(ctx, notifier)

	settingsViewModel := viewmodel.NewSettingsViewModel(
		fyneSettings,
		notifier,
		appState,
		eventBus)
	connectionViewModel := viewmodel.NewConnectionViewModel(
		connectionsRepository,
		settingsViewModel,
		appState,
		notifier,
		eventBus,
	)
	explorerViewModel := viewmodel.NewExplorerViewModel(
		settingsViewModel,
		notifier,
		connectionViewModel.Deck().SelectedConnection(),
		eventBus,
		appState,
	)

	editorViewModel := viewmodel.NewEditorViewModel(eventBus, notifier,
		connectionViewModel.Deck().SelectedConnection())

	appCtx := appcontext.New(
		appName,
		w,
		explorerViewModel,
		connectionViewModel,
		settingsViewModel,
		notificationsViewModel,
		editorViewModel,
		initRoute,
		appViews,
		logger,
		fyneSettings,
		eventBus,
		appState,
	)

	var one sync.Once
	w.SetOnClosed(func() {
		one.Do(func() {
			cancel()
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

type notifierAdapter struct {
	event.NopNotifier
	notifier notification.Repository
}

func (n *notifierAdapter) NotifyPublished(evt event.Event) {
	content, err := json.MarshalIndent(evt, "", "  ")
	if err != nil {
		n.notifier.NotifyError(err)
		return
	}
	title := fmt.Sprintf("Event published: %s", evt.Type())
	n.notifier.NotifyDebug(title, string(content))
}
