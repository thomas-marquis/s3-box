package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
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

		dir := tu.MakeDirectory(t, "test",
			tu.WithRootParent(),
			tu.IsLoaded())

		st := state.New()
		deck := connection_deck.New()
		st.Connection().Init(deck)
		mockAppCtx.EXPECT().State().Return(st).AnyTimes()

		// When
		res := widget.NewDirectoryDetails(mockAppCtx)
		res.Select(dir)
		w := fyne_test.NewWindow(res)
		w.Resize(fyne.NewSize(600, 400))
		c := w.Canvas()

		// Then
		tu.AssertImageMatches(t, "images/directory-details.png", c.Capture())
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

		st := state.New()
		deck := connection_deck.New()
		conn := deck.New("Conn 1", "ak1", "sk1", "b1",
			connection_deck.WithReadOnlyOption(true)).
			Payload().(connection_deck.CreateConnectionTriggered).Connection()
		u.SkipV(deck.Select(conn.ID()))
		st.Connection().Init(deck)
		mockAppCtx.EXPECT().State().Return(st).AnyTimes()

		dir := tu.MakeDirectory(t, "test",
			tu.WithParent("home", tu.WithRootParent()),
			tu.IsLoaded(),
			tu.WithConnectionId(conn.ID()))

		// When
		res := widget.NewDirectoryDetails(mockAppCtx)
		res.Select(dir)
		w := fyne_test.NewWindow(res)
		w.Resize(fyne.NewSize(600, 400))
		c := w.Canvas()

		// Then
		tu.AssertImageMatches(t, "images/directory-details-read-only.png", c.Capture())
	})
}
