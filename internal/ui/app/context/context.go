package appcontext

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"

	"fyne.io/fyne/v2"
	"go.uber.org/zap"
)

type AppContext interface {
	Navigate(route navigation.Route) (*fyne.Container, error)
	CurrentRoute() navigation.Route

	ExplorerViewModel() viewmodel.ExplorerViewModel
	ConnectionViewModel() viewmodel.ConnectionViewModel
	SettingsViewModel() viewmodel.SettingsViewModel
	NotificationViewModel() viewmodel.NotificationViewModel

	Window() fyne.Window
	L() *zap.Logger
	FyneSettings() fyne.Settings
	AppContent() fyne.CanvasObject

	Bus() event.Bus
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

	currentRoute navigation.Route
	menu         map[navigation.Route]Menu

	bus event.Bus

	mainWidget *AppWidget
}

var _ AppContext = &AppContextImpl{}

func New(
	appName string,
	window fyne.Window,
	explorerViewModel viewmodel.ExplorerViewModel,
	connectionViewModel viewmodel.ConnectionViewModel,
	settingsViewModel viewmodel.SettingsViewModel,
	notificationViewModel viewmodel.NotificationViewModel,
	initialRoute navigation.Route,
	menu map[navigation.Route]Menu,
	logger *zap.Logger,
	settings fyne.Settings,
	bus event.Bus,
) *AppContextImpl {

	menuList := make([]Menu, len(menu))
	for _, menu := range menu {
		menuList[menu.Index] = menu
	}

	ctx := &AppContextImpl{
		explorerViewModel:     explorerViewModel,
		connectionsViewModel:  connectionViewModel,
		settingsViewModel:     settingsViewModel,
		notificationViewModel: notificationViewModel,
		window:                window,
		logger:                logger,
		currentRoute:          initialRoute,
		fyneSettings:          settings,
		bus:                   bus,
		menu:                  menu,
	}

	ctx.mainWidget = newAppWidget(appName, menuList, ctx.Navigate)

	return ctx
}

func (ctx *AppContextImpl) AppContent() fyne.CanvasObject {
	return ctx.mainWidget
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

func (ctx *AppContextImpl) Navigate(route navigation.Route) (*fyne.Container, error) {
	if _, ok := ctx.menu[route]; !ok {
		return nil, navigation.ErrRouteNotFound
	}

	view, err := ctx.menu[route].View(ctx)
	if err != nil {
		return nil, err
	}

	ctx.mainWidget.SetViewContent(view)
	ctx.currentRoute = route

	return view, nil
}

func (ctx *AppContextImpl) CurrentRoute() navigation.Route {
	return ctx.currentRoute
}

func (ctx *AppContextImpl) Bus() event.Bus {
	return ctx.bus
}
