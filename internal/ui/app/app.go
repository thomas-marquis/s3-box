package app

import (
	"context"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views"
	"time"

	"fyne.io/fyne/v2"
	fyne_app "fyne.io/fyne/v2/app"
	"go.uber.org/zap"
)

const (
	appId = "fr.peaksys.go2s3"
)

type Go2S3App struct {
	ctx       appcontext.AppContext
	views     map[navigation.Route]func(appcontext.AppContext) (*fyne.Container, error)
	initRoute navigation.Route
}

func New(logger *zap.Logger, initRoute navigation.Route) (*Go2S3App, error) {
	appViews := make(map[navigation.Route]func(appcontext.AppContext) (*fyne.Container, error))
	appViews[navigation.ExplorerRoute] = views.GetFileExplorerView
	appViews[navigation.ConnectionRoute] = views.GetConnectionView

	sugarLog := logger.Sugar()
	a := fyne_app.NewWithID(appId)
	w := a.NewWindow("S3 Box")

	connRepo := infrastructure.NewConnectionRepositoryImpl(a.Preferences())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // TODO get this from the user's settings
	defer cancel()
	lastSelectedConn, err := connRepo.GetSelectedConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		sugarLog.Error("Error getting selected connection", err)
		return nil, err
	}
	explRepo, err := infrastructure.NewExplorerRepositoryImpl(logger, lastSelectedConn)
	if err != nil {
		sugarLog.Error("Error creating explorer repository", err)
		return nil, err
	}

	dirSvc := explorer.NewDirectoryService(explRepo)
	vm := viewmodel.NewViewModel(explRepo, dirSvc, connRepo)
	appctx := appcontext.New(w, vm, initRoute, appViews, logger)

	w.SetOnClosed(func() {
		close(appctx.ExitChan())
	})

	w.SetMainMenu(getMainMenu(appctx))

	return &Go2S3App{
		views:     appViews,
		initRoute: initRoute,
		ctx:       appctx,
	}, nil
}

func (a *Go2S3App) Start() error {
	a.ctx.W().Resize(fyne.NewSize(1000, 700))
	err := a.ctx.Navigate(a.initRoute)
	if err != nil {
		return err
	}
	a.ctx.W().ShowAndRun()
	return nil
}

func getMainMenu(ctx appcontext.AppContext) *fyne.MainMenu {
	settingsMenu := fyne.NewMenu("Settings",
		fyne.NewMenuItem("Manage connections", func() {
			ctx.Navigate(navigation.ConnectionRoute)
		}),
	)
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Explorer view", func() {
			ctx.Navigate(navigation.ExplorerRoute)
		}),
	)
	return fyne.NewMainMenu(fileMenu, settingsMenu)
}
