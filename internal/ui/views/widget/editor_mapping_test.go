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
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

// uv run ./tools/diff_images.py --folders internal/ui/views/widget/testdata/images internal/ui/views/widget/testdata/failed/images --color "red"

func TestEditorMapping(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should display file matchers table with data", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
		mockSettingsVM := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

		textFactory := mock_editor.NewMockFactory(ctrl)
		textFactory.EXPECT().DisplayLabel().Return("Text Editor").AnyTimes()
		textFactory.EXPECT().Name().Return("text").AnyTimes()

		csvFactory := mock_editor.NewMockFactory(ctrl)
		csvFactory.EXPECT().DisplayLabel().Return("CSV Editor").AnyTimes()
		csvFactory.EXPECT().Name().Return("csv").AnyTimes()

		mockAppCtx.EXPECT().SettingsViewModel().Return(mockSettingsVM).AnyTimes()
		mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

		st := state.New()
		deck := connection_deck.New()
		st.Connection().Init(deck)

		selector := st.Editors().Selector()
		selector.RegisterEditor(textFactory)
		selector.RegisterEditor(csvFactory)

		u.Skip(selector.RegisterMapping("csv", "\\.csv$"))
		u.Skip(selector.RegisterMapping("text", "\\.(txt|md)$"))

		mockAppCtx.EXPECT().State().Return(st).AnyTimes()

		// When
		res := widget.NewEditorMappingsTable(mockAppCtx)
		w := fyne_test.NewWindow(res)
		w.Resize(fyne.NewSize(600, 400))
		c := w.Canvas()
		res.Refresh()

		// Then
		tu.AssertImageMatches(t, "images/file-matchers-table.png", c.Capture())
	})
}
