package appcontext

import (
	"go2s3/internal/ui/app/navigation"
	"go2s3/internal/ui/viewmodel"

	"fyne.io/fyne/v2"
	"go.uber.org/zap"
)

type AppContext interface {
	Navigate(route navigation.Route) error
	CurrentRoute() navigation.Route
	Vm() *viewmodel.ViewModel
	W() fyne.Window
	L() *zap.Logger
	ExitChan() chan struct{}
}

type AppContextImpl struct {
	vm       *viewmodel.ViewModel
	w        fyne.Window
	l        *zap.Logger
	exitChan chan struct{}

	currentRoute navigation.Route
	views        map[navigation.Route]func(AppContext) (*fyne.Container, error)
}

func New(
	w fyne.Window,
	vm *viewmodel.ViewModel,
	initialRoute navigation.Route,
	views map[navigation.Route]func(AppContext) (*fyne.Container, error),
	logger *zap.Logger,
) *AppContextImpl {
	return &AppContextImpl{
		vm:           vm,
		w:            w,
		l:            logger,
		exitChan:     make(chan struct{}),
		currentRoute: initialRoute,
		views:        views,
	}
}

func (ctx *AppContextImpl) Vm() *viewmodel.ViewModel {
	return ctx.vm
}

func (ctx *AppContextImpl) W() fyne.Window {
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

	ctx.W().SetContent(view)
	ctx.currentRoute = route

	return nil
}

func (ctx *AppContextImpl) CurrentRoute() navigation.Route {
	return ctx.currentRoute
}
