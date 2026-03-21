package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

func TestDirectoryDetails(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should display directory details", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
		mockExplorerVM := mocks_viewmodel.NewMockExplorerViewModel(ctrl)
		mockConnVM := mocks_viewmodel.NewMockConnectionViewModel(ctrl)

		mockAppCtx.EXPECT().ExplorerViewModel().Return(mockExplorerVM).AnyTimes()
		mockAppCtx.EXPECT().ConnectionViewModel().Return(mockConnVM).AnyTimes()
		mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

		fakeIsLoadingBinding := binding.NewBool()
		mockExplorerVM.EXPECT().IsSelectedDirectoryLoading().Return(fakeIsLoadingBinding).AnyTimes()

		dir := testutil.FakeNotLoadedRootDirectory(t)

		mockConnVM.EXPECT().IsReadOnly().Return(false)

		// When
		res := widget.NewDirectoryDetails(mockAppCtx)
		res.Select(dir)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "directory_details", c)
	})

	t.Run("should display directory details in read-only mode", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
		mockExplorerVM := mocks_viewmodel.NewMockExplorerViewModel(ctrl)
		mockConnVM := mocks_viewmodel.NewMockConnectionViewModel(ctrl)

		mockAppCtx.EXPECT().ExplorerViewModel().Return(mockExplorerVM).AnyTimes()
		mockAppCtx.EXPECT().ConnectionViewModel().Return(mockConnVM).AnyTimes()
		mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

		fakeIsLoadingBinding := binding.NewBool()
		mockExplorerVM.EXPECT().IsSelectedDirectoryLoading().Return(fakeIsLoadingBinding).AnyTimes()

		dir := testutil.FakeNotLoadedRootDirectory(t)

		mockConnVM.EXPECT().IsReadOnly().Return(true)

		// When
		res := widget.NewDirectoryDetails(mockAppCtx)
		res.Select(dir)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "directory_details_readonly", c)
	})
}
