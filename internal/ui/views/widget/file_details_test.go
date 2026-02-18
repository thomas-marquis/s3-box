package widget_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

const (
	fakeFileSizeLimitKB = 2048
)

type fileDetailsMocks struct {
	mockAppCtx     *mocks_appcontext.MockAppContext
	mockExplorerVM *mocks_viewmodel.MockExplorerViewModel
	mockConnVM     *mocks_viewmodel.MockConnectionViewModel
	mockSettingsVM *mocks_viewmodel.MockSettingsViewModel
	mockEditorVM   *mocks_viewmodel.MockEditorViewModel

	sizeLimitBinding binding.Int
}

func setupFileDetailsMocks(t *testing.T) fileDetailsMocks {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := fileDetailsMocks{
		mockAppCtx:       mocks_appcontext.NewMockAppContext(ctrl),
		mockExplorerVM:   mocks_viewmodel.NewMockExplorerViewModel(ctrl),
		mockConnVM:       mocks_viewmodel.NewMockConnectionViewModel(ctrl),
		mockSettingsVM:   mocks_viewmodel.NewMockSettingsViewModel(ctrl),
		mockEditorVM:     mocks_viewmodel.NewMockEditorViewModel(ctrl),
		sizeLimitBinding: binding.NewInt(),
	}

	m.mockAppCtx.EXPECT().ExplorerViewModel().Return(m.mockExplorerVM).AnyTimes()
	m.mockAppCtx.EXPECT().ConnectionViewModel().Return(m.mockConnVM).AnyTimes()
	m.mockAppCtx.EXPECT().SettingsViewModel().Return(m.mockSettingsVM).AnyTimes()
	m.mockAppCtx.EXPECT().EditorViewModel().Return(m.mockEditorVM).AnyTimes()
	m.mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

	m.sizeLimitBinding.Set(fakeFileSizeLimitKB) // nolint:errcheck
	m.mockSettingsVM.EXPECT().FileSizeLimitKB().Return(m.sizeLimitBinding).AnyTimes()

	return m
}

func TestFileDetails(t *testing.T) {
	fyne_test.NewApp()

	lastModified := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	file, _ := directory.NewFile("test.txt", directory.RootPath,
		directory.WithFileSize(fakeFileSizeLimitKB),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should display file details", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocks(t)
		m.mockConnVM.EXPECT().IsReadOnly().Return(false).AnyTimes()
		m.mockSettingsVM.EXPECT().CurrentFileSizeLimitBytes().Return(fakeFileSizeLimitKB)

		// When
		res := widget.NewFileDetails(m.mockAppCtx)
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "file_details", c)
	})

	t.Run("should disable preview if file is too large", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocks(t)
		m.mockConnVM.EXPECT().IsReadOnly().Return(false).AnyTimes()
		m.mockSettingsVM.EXPECT().CurrentFileSizeLimitBytes().Return(512)

		// When
		res := widget.NewFileDetails(m.mockAppCtx)
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "file_details_large", c)
	})

	t.Run("should disable delete if read-only", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocks(t)
		m.mockConnVM.EXPECT().IsReadOnly().Return(true).AnyTimes()
		m.mockSettingsVM.EXPECT().CurrentFileSizeLimitBytes().Return(2048)

		// When
		res := widget.NewFileDetails(m.mockAppCtx)
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "file_details_readonly", c)
	})
}
