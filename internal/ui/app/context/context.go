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
	Vm() *viewmodel.ViewModel
	ExplorerVM() *viewmodel.ExplorerViewModel
	W() fyne.Window
	Log() *zap.Logger
	ExitChan() chan struct{}
}

type AppContextImpl struct {
	evm      *viewmodel.ExplorerViewModel
	legacyVm *viewmodel.ViewModel
	w        fyne.Window
	logger   *zap.Logger
	exitChan chan struct{}

	currentRoute navigation.Route
	views        map[navigation.Route]func(AppContext) (*fyne.Container, error)
}

var _ AppContext = &AppContextImpl{}

func New(
	w fyne.Window,
	evm *viewmodel.ExplorerViewModel,
	legacyVm *viewmodel.ViewModel,
	initialRoute navigation.Route,
	views map[navigation.Route]func(AppContext) (*fyne.Container, error),
	logger *zap.Logger,
) *AppContextImpl {
	return &AppContextImpl{
		evm:          evm,
		legacyVm:     legacyVm,
		w:            w,
		logger:       logger,
		exitChan:     make(chan struct{}),
		currentRoute: initialRoute,
		views:        views,
	}
}

func (ctx *AppContextImpl) Vm() *viewmodel.ViewModel {
	return ctx.legacyVm
}

func (ctx *AppContextImpl) ExplorerVM() *viewmodel.ExplorerViewModel {
	return ctx.evm
}

func (ctx *AppContextImpl) W() fyne.Window {
	return ctx.w
}

func (ctx *AppContextImpl) Log() *zap.Logger {
	return ctx.logger
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

	ctx.W().SetContent(view)
	ctx.currentRoute = route

	return nil
}

func (ctx *AppContextImpl) CurrentRoute() navigation.Route {
	return ctx.currentRoute
}
