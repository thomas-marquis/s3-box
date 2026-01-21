package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
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
	conn1 := deck.New("Conn 1", "ak1", "sk1", "b1").Connection()
	conn2 := deck.New("Conn 2", "ak2", "sk2", "b2").Connection()

	connections := binding.NewUntypedList()
	_ = connections.Append(conn1)
	_ = connections.Append(conn2)

	mockAppCtx.EXPECT().ConnectionViewModel().Return(mockConnVM).AnyTimes()
	mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()
	mockConnVM.EXPECT().Connections().Return(connections).AnyTimes()
	mockConnVM.EXPECT().Deck().Return(deck).AnyTimes()

	t.Run("should display list of connections", func(t *testing.T) {
		// When
		res := widget.NewConnectionList(mockAppCtx)
		c := fyne_test.NewWindow(res).Canvas()
		res.Refresh()

		// Then
		fyne_test.AssertRendersToMarkup(t, "connection_list", c)
	})
}
