package widget_test

import (
	"testing"
	"time"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/values"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

const (
	fakeFileSizeLimitKB = 2048
)

var (
	lastModified = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
)

type fileDetailsMocks struct {
	mockAppCtx     *mocks_appcontext.MockAppContext
	mockExplorerVM *mocks_viewmodel.MockExplorerViewModel
	mockConnVM     *mocks_viewmodel.MockConnectionViewModel
	mockSettingsVM *mocks_viewmodel.MockSettingsViewModel
	mockEditorVM   *mocks_viewmodel.MockEditorViewModel
	mockTagsVM     *mocks_viewmodel.MockTagsViewModel
	mockState      *state.State
}

func setupFileDetailsMocks(t *testing.T) fileDetailsMocks {
	return setupFileDetailsMocksWithLimit(t, 20*1024)
}

func setupFileDetailsMocksWithLimit(t *testing.T, limitBytes uint64) fileDetailsMocks {
	t.Helper()

	ctrl := gomock.NewController(t)

	st := state.New()
	deck := connection_deck.New()
	st.Connection().Init(deck)

	m := fileDetailsMocks{
		mockAppCtx:     mocks_appcontext.NewMockAppContext(ctrl),
		mockExplorerVM: mocks_viewmodel.NewMockExplorerViewModel(ctrl),
		mockConnVM:     mocks_viewmodel.NewMockConnectionViewModel(ctrl),
		mockSettingsVM: mocks_viewmodel.NewMockSettingsViewModel(ctrl),
		mockEditorVM:   mocks_viewmodel.NewMockEditorViewModel(ctrl),
		mockTagsVM:     mocks_viewmodel.NewMockTagsViewModel(ctrl),
		mockState:      st,
	}

	m.mockTagsVM.EXPECT().Select(gomock.Any()).AnyTimes()

	m.mockAppCtx.EXPECT().ExplorerViewModel().Return(m.mockExplorerVM).AnyTimes()
	m.mockAppCtx.EXPECT().ConnectionViewModel().Return(m.mockConnVM).AnyTimes()
	m.mockAppCtx.EXPECT().SettingsViewModel().Return(m.mockSettingsVM).AnyTimes()
	m.mockAppCtx.EXPECT().EditorViewModel().Return(m.mockEditorVM).AnyTimes()
	m.mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()
	m.mockAppCtx.EXPECT().State().Return(m.mockState).AnyTimes()
	m.mockAppCtx.EXPECT().TagsViewModel().Return(m.mockTagsVM).AnyTimes()

	// Register the settings that file_details needs
	u.Skip(m.mockState.Settings().Get().Notify(event.New(settings.WriteSucceeded{
		Name:  values.SettingEditFileSizeLimitByte,
		Value: limitBytes,
	})))

	return m
}

func TestFileDetails(t *testing.T) {
	fyne_test.NewApp()

	var file *directory.File
	tu.MakeDirectory(t, "", tu.AsRoot(),
		tu.WithFileTo("test.txt", &file,
			directory.WithFileSize(fakeFileSizeLimitKB),
			directory.WithFileLastModified(lastModified)))

	t.Run("should display file details", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocks(t)

		// When
		res := widget.NewFileDetails(m.mockAppCtx)
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		tu.AssertImageMatches(t, "images/file-details.png", c.Capture())
	})

	t.Run("should disable edit button when file is too large", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocksWithLimit(t, 512)

		// When
		res := widget.NewFileDetails(m.mockAppCtx)
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		tu.AssertImageMatches(t, "images/file-details-too-large.png", c.Capture())
	})

	t.Run("should disable delete if read-only", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocks(t)

		deck := m.mockAppCtx.State().Connection().Deck()
		conn := deck.New("Conn 1", "ak1", "sk1", "b1",
			connection_deck.WithReadOnlyOption(true)).
			Payload().(connection_deck.CreateConnectionTriggered).Connection()
		u.SkipV(deck.Select(conn.ID()))

		// When
		res := widget.NewFileDetails(m.mockAppCtx)
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		tu.AssertImageMatches(t, "images/file-details-read-only.png", c.Capture())
	})
}
