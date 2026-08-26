package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

func TestConnectionList(t *testing.T) {
	fyne_test.NewApp()

	ctrl := gomock.NewController(t)
	mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
	mockConnVM := mocks_viewmodel.NewMockConnectionViewModel(ctrl)

	deck := connection_deck.New()
	conn1 := deck.New("Conn 1", "ak1", "sk1", "b1").
		Payload().(connection_deck.CreateConnectionTriggered).Connection()
	conn2 := deck.New("Conn 2", "ak2", "sk2", "b2").
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	connections := binding.NewList[*connection_deck.Connection](connection_deck.Compare)
	_ = connections.Append(conn1)
	_ = connections.Append(conn2)

	mockAppCtx.EXPECT().ConnectionViewModel().Return(mockConnVM).AnyTimes()
	mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()
	st := state.New()
	st.Connection().Init(deck)
	mockAppCtx.EXPECT().State().Return(st).AnyTimes()

	t.Run("should display list of connections", func(t *testing.T) {
		// When
		res := widget.NewConnectionList(mockAppCtx)
		w := fyne_test.NewWindow(res)
		w.Resize(fyne.NewSize(400, 300))
		c := w.Canvas()
		res.Refresh()

		// Then
		tu.AssertImageMatches(t, "images/connection-list.png", c.Capture())
	})
}
