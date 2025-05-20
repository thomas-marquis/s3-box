package views_test

import (
	"context"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/settings"
	"github.com/thomas-marquis/s3-box/internal/ui/app"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_settings "github.com/thomas-marquis/s3-box/mocks/settings"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func Test_GetSettingsView_ShouldBuildViewWithoutError(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

	// Setup settings
	settingsRepo := mocks_settings.NewMockRepository(ctrl)
	fakeSettings, _ := settings.NewSettings(10, 2)
	settingsRepo.EXPECT().
		Get(gomock.AssignableToTypeOf(ctxType)).
		Return(fakeSettings, nil).
		Times(1)

	// Setup connection
	connRepo := mocks_connection.NewMockRepository(ctrl)

	fakeConnection := connection.NewConnection("demo", "AZERTY", "123456", "MyBucket", connection.AsAWSConnection("eu-west-3"))
	connRepo.EXPECT().
		ListConnections(gomock.AssignableToTypeOf(ctxType)).
		Return([]*connection.Connection{fakeConnection}, nil).
		Times(1)

	connRepo.EXPECT().
		GetSelectedConnection(gomock.AssignableToTypeOf(ctxType)).
		Return(fakeConnection, nil).
		Times(2)

	connRepo.EXPECT().
		GetByID(gomock.AssignableToTypeOf(ctxType), gomock.Eq(fakeConnection.ID)).
		Return(fakeConnection, nil).
		Times(1)

	fakeApp := fyne_test.NewTempApp(t)
	fakeWindow := fakeApp.NewWindow("Test")

	testViews := make(map[navigation.Route]func(appcontext.AppContext) (*fyne.Container, error))
	testViews[navigation.SettingsRoute] = views.GetSettingsView

	appCtx := app.BuildAppContext(
		connRepo,
		settingsRepo,
		zap.NewExample(),
		fakeConnection,
		fakeWindow,
		navigation.SettingsRoute,
		testViews,
		fakeApp.Settings(),
	)

	// When
	v, err := views.GetSettingsView(appCtx)

	// Then
	assert.NoError(t, err, "no error should be returned")
	assert.NotNil(t, v, "settingsView should not be nil")

	// Assert structure
	var ok bool
	contentBloc, ok := v.Objects[1].(*fyne.Container)
	assert.True(t, ok, "Content block should be a fyne.Container")

	form, ok := contentBloc.Objects[0].(*widget.Form)
	assert.True(t, ok, "Form should be a widget.Form")

	formItems := form.Items
	assert.Len(t, formItems, 3, "Invalid number of form items")
}
