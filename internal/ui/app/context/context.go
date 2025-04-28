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
	ExplorerViewModel() *viewmodel.ExplorerViewModel
	ConnectionViewModel() *viewmodel.ConnectionViewModel
	SettingsViewModel() viewmodel.SettingsViewModel
	Window() fyne.Window
	L() *zap.Logger
	ExitChan() chan struct{}
}

type AppContextImpl struct {
	vm       *viewmodel.ExplorerViewModel
	connVm   *viewmodel.ConnectionViewModel
	settingsVm viewmodel.SettingsViewModel
	w        fyne.Window
	l        *zap.Logger
	exitChan chan struct{}

	currentRoute navigation.Route
	views        map[navigation.Route]func(AppContext) (*fyne.Container, error)
}

var _ AppContext = &AppContextImpl{}

func New(
	w fyne.Window,
	vm *viewmodel.ExplorerViewModel,
	connVm *viewmodel.ConnectionViewModel,
	settingsVm viewmodel.SettingsViewModel,
	initialRoute navigation.Route,
	views map[navigation.Route]func(AppContext) (*fyne.Container, error),
	logger *zap.Logger,
) *AppContextImpl {
	return &AppContextImpl{
		vm:           vm,
		connVm:       connVm,
		settingsVm:   settingsVm,
		w:            w,
		l:            logger,
		exitChan:     make(chan struct{}),
		currentRoute: initialRoute,
		views:        views,
	}
}

func (ctx *AppContextImpl) ExplorerViewModel() *viewmodel.ExplorerViewModel {
	return ctx.vm
}

func (ctx *AppContextImpl) ConnectionViewModel() *viewmodel.ConnectionViewModel {
	return ctx.connVm
}

func (ctx *AppContextImpl) SettingsViewModel() viewmodel.SettingsViewModel {
	return ctx.settingsVm
}

func (ctx *AppContextImpl) Window() fyne.Window {
	return ctx.w
}

func (ctx *AppContextImpl) L() *zap.Logger {
	return ctx.l
}

func (ctx *AppContextImpl) ExitChan() chan struct{} {
	return ctx.exitChan
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
