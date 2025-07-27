package appcontext

import (
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"

	"fyne.io/fyne/v2"
	"go.uber.org/zap"
)

type AppContext interface {
	Navigate(route navigation.Route) error
	CurrentRoute() navigation.Route

	ExplorerViewModel() viewmodel.ExplorerViewModel
	ConnectionViewModel() viewmodel.ConnectionViewModel
	SettingsViewModel() viewmodel.SettingsViewModel
	NotificationViewModel() viewmodel.NotificationViewModel

	Window() fyne.Window
	L() *zap.Logger
	FyneSettings() fyne.Settings

	Terminate()
	SubscribeTerminate(chan struct{})
}

type AppContextImpl struct {
	explorerViewModel     viewmodel.ExplorerViewModel
	connectionsViewModel  viewmodel.ConnectionViewModel
	settingsViewModel     viewmodel.SettingsViewModel
	notificationViewModel viewmodel.NotificationViewModel

	window       fyne.Window
	logger       *zap.Logger
	exitChan     chan struct{}
	fyneSettings fyne.Settings

	currentRoute         navigation.Route
	views                map[navigation.Route]View
	terminateSubscribers []chan struct{}
}

var _ AppContext = &AppContextImpl{}

func New(
	window fyne.Window,
	explorerViewModel viewmodel.ExplorerViewModel,
	connectionViewModel viewmodel.ConnectionViewModel,
	settingsViewModel viewmodel.SettingsViewModel,
	notificationViewModel viewmodel.NotificationViewModel,
	initialRoute navigation.Route,
	views map[navigation.Route]View,
	logger *zap.Logger,
	settings fyne.Settings,
) *AppContextImpl {
	return &AppContextImpl{
		explorerViewModel:     explorerViewModel,
		connectionsViewModel:  connectionViewModel,
		settingsViewModel:     settingsViewModel,
		notificationViewModel: notificationViewModel,
		window:                window,
		logger:                logger,
		currentRoute:          initialRoute,
		views:                 views,
		fyneSettings:          settings,
		terminateSubscribers:  make([]chan struct{}, 0),
	}
}

func (ctx *AppContextImpl) FyneSettings() fyne.Settings {
	return ctx.fyneSettings
}

func (ctx *AppContextImpl) ExplorerViewModel() viewmodel.ExplorerViewModel {
	return ctx.explorerViewModel
}

func (ctx *AppContextImpl) ConnectionViewModel() viewmodel.ConnectionViewModel {
	return ctx.connectionsViewModel
}

func (ctx *AppContextImpl) SettingsViewModel() viewmodel.SettingsViewModel {
	return ctx.settingsViewModel
}

func (ctx *AppContextImpl) NotificationViewModel() viewmodel.NotificationViewModel {
	return ctx.notificationViewModel
}

func (ctx *AppContextImpl) Window() fyne.Window {
	return ctx.window
}

func (ctx *AppContextImpl) L() *zap.Logger {
	return ctx.logger
}

func (ctx *AppContextImpl) Terminate() {
	for _, subscriber := range ctx.terminateSubscribers {
		subscriber <- struct{}{}
	}
}

func (ctx *AppContextImpl) Navigate(route navigation.Route) error {
	if _, ok := ctx.views[route]; !ok {
		return navigation.ErrRouteNotFound
	}

	view, err := ctx.views[route](ctx)
	if err != nil {
		return err
	}

	ctx.Window().SetContent(view)
	ctx.currentRoute = route

	return nil
}

func (ctx *AppContextImpl) CurrentRoute() navigation.Route {
	return ctx.currentRoute
}

func (ctx *AppContextImpl) SubscribeTerminate(subscriber chan struct{}) {
	ctx.terminateSubscribers = append(ctx.terminateSubscribers, subscriber)
}
