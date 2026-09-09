package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

func TestFileMatchersTable(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should display file matchers table with data", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
		mockSettingsVM := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

		fakeFactory := func(bus event.Bus, window fyne.Window, file *directory.File) editor.Editor {
			return mock_editor.NewMockEditor(ctrl)
		}

		textSelector := &editor.Selector{
			Name:    "text",
			Factory: fakeFactory,
			Pattern: ".*\\.(txt|md)$",
		}

		csvSelector := &editor.Selector{
			Name:    "csv",
			Factory: fakeFactory,
			Pattern: ".*\\.(csv|tsv)$",
		}

		mockAppCtx.EXPECT().SettingsViewModel().Return(mockSettingsVM).AnyTimes()
		mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

		st := state.New()
		deck := connection_deck.New()
		st.Connection().Init(deck)
		st.Settings().EditorSelectors().Set([]*editor.Selector{textSelector, csvSelector})
		mockAppCtx.EXPECT().State().Return(st).AnyTimes()

		// When
		res := widget.NewEditorSelectorTable(mockAppCtx)
		w := fyne_test.NewWindow(res)
		w.Resize(fyne.NewSize(600, 400))
		c := w.Canvas()
		res.Refresh()

		// Then
		tu.AssertImageMatches(t, "images/file-matchers-table.png", c.Capture())
	})
}
