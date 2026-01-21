package widget_test

import (
	"testing"
	"time"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

func TestFileDetails(t *testing.T) {
	fyne_test.NewApp()

	ctrl := gomock.NewController(t)
	mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
	mockExplorerVM := mocks_viewmodel.NewMockExplorerViewModel(ctrl)
	mockConnVM := mocks_viewmodel.NewMockConnectionViewModel(ctrl)
	mockSettingsVM := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockAppCtx.EXPECT().ExplorerViewModel().Return(mockExplorerVM).AnyTimes()
	mockAppCtx.EXPECT().ConnectionViewModel().Return(mockConnVM).AnyTimes()
	mockAppCtx.EXPECT().SettingsViewModel().Return(mockSettingsVM).AnyTimes()
	mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

	lastModified := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	file, _ := directory.NewFile("test.txt", directory.RootPath,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should display file details", func(t *testing.T) {
		// Given
		mockConnVM.EXPECT().IsReadOnly().Return(false)
		mockSettingsVM.EXPECT().CurrentMaxFilePreviewSizeBytes().Return(2048)

		// When
		res := widget.NewFileDetails(mockAppCtx)
		res.Render(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "file_details", c)
	})

	t.Run("should disable preview if file is too large", func(t *testing.T) {
		// Given
		mockConnVM.EXPECT().IsReadOnly().Return(false)
		mockSettingsVM.EXPECT().CurrentMaxFilePreviewSizeBytes().Return(512)

		// When
		res := widget.NewFileDetails(mockAppCtx)
		res.Render(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "file_details_large", c)
	})

	t.Run("should disable delete if read-only", func(t *testing.T) {
		// Given
		mockConnVM.EXPECT().IsReadOnly().Return(true)
		mockSettingsVM.EXPECT().CurrentMaxFilePreviewSizeBytes().Return(2048)

		// When
		res := widget.NewFileDetails(mockAppCtx)
		res.Render(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "file_details_readonly", c)
	})
}
