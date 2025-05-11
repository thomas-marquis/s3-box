package views_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/app"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"
	mocks_settings "github.com/thomas-marquis/s3-box/mocks/settings"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func Test_SettingView_ShouldUpdateAndSaveSettings(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)

	settingsRepo := mocks_settings.NewMockRepository(ctrl)
	directoryRepo := mocks_explorer.NewMockS3DirectoryRepository(ctrl)
	fileRepo := mocks_explorer.NewMockS3FileRepository(ctrl)
	connRepo := mocks_connection.NewMockRepository(ctrl)

	lastConnection := connection.NewConnection(
		"test_connection",
		"localhost",
		"access_key",
		"secret_key",
		"myBucket",
		false,
		"",
	)

	fakeApp := fyne_test.NewTempApp(t)
	fakeWindow := fakeApp.NewWindow("Test")

	testViews := make(map[navigation.Route]func(appcontext.AppContext) (*fyne.Container, error))
	testViews[navigation.SettingsRoute] = views.GetSettingsView

	appCtx := app.BuildAppContext(
		connRepo,
		settingsRepo,
		func(ctx context.Context, connID uuid.UUID) (explorer.S3DirectoryRepository, error) {
			return directoryRepo, nil
		},
		func(ctx context.Context, connID uuid.UUID) (explorer.S3FileRepository, error) {
			return fileRepo, nil
		},
		zap.NewExample(),
		lastConnection,
		fakeWindow,
		navigation.SettingsRoute,
		testViews,
		fakeApp.Settings(),
	)
	settingsView, _ := views.GetSettingsView(appCtx)

	// When
	var _ = settingsView.Objects[1].(*widget.Form) // TODO: finish to implement the test
}
