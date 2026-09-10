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
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

const (
	fakeFileSizeLimitKB = 2048
)

var (
	lastModified = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
)

type fileDetailsFixture struct {
	t              *testing.T
	ctrl           *gomock.Controller
	limitBytes     uint64
	mockAppCtx     *mocks_appcontext.MockAppContext
	mockExplorerVM *mocks_viewmodel.MockExplorerViewModel
	mockConnVM     *mocks_viewmodel.MockConnectionViewModel
	mockSettingsVM *mocks_viewmodel.MockSettingsViewModel
	mockEditorVM   *mocks_viewmodel.MockEditorViewModel
	mockTagsVM     *mocks_viewmodel.MockTagsViewModel
	mockState      *state.State
	mockFactory    *mock_editor.MockFactory
	file           *directory.File
}

func newFileDetailsFixture(t *testing.T, limitBytes uint64) *fileDetailsFixture {
	t.Helper()

	f := &fileDetailsFixture{
		t:          t,
		ctrl:       gomock.NewController(t),
		limitBytes: limitBytes,
		mockState:  state.New(),
	}

	deck := connection_deck.New()
	f.mockState.Connection().Init(deck)

	f.mockAppCtx = mocks_appcontext.NewMockAppContext(f.ctrl)
	f.mockExplorerVM = mocks_viewmodel.NewMockExplorerViewModel(f.ctrl)
	f.mockConnVM = mocks_viewmodel.NewMockConnectionViewModel(f.ctrl)
	f.mockSettingsVM = mocks_viewmodel.NewMockSettingsViewModel(f.ctrl)
	f.mockEditorVM = mocks_viewmodel.NewMockEditorViewModel(f.ctrl)
	f.mockTagsVM = mocks_viewmodel.NewMockTagsViewModel(f.ctrl)

	f.mockTagsVM.EXPECT().Select(gomock.Any()).AnyTimes()

	f.mockAppCtx.EXPECT().ExplorerViewModel().Return(f.mockExplorerVM).AnyTimes()
	f.mockAppCtx.EXPECT().ConnectionViewModel().Return(f.mockConnVM).AnyTimes()
	f.mockAppCtx.EXPECT().SettingsViewModel().Return(f.mockSettingsVM).AnyTimes()
	f.mockAppCtx.EXPECT().EditorViewModel().Return(f.mockEditorVM).AnyTimes()
	f.mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()
	f.mockAppCtx.EXPECT().State().Return(f.mockState).AnyTimes()
	f.mockAppCtx.EXPECT().TagsViewModel().Return(f.mockTagsVM).AnyTimes()

	// Register the settings that file_details needs
	u.Skip(f.mockState.Settings().Get().Notify(event.New(settings.WriteSucceeded{
		Name:  values.SettingEditFileSizeLimitByte,
		Value: f.limitBytes,
	})))

	f.mockFactory = mock_editor.NewMockFactory(f.ctrl)
	f.mockFactory.EXPECT().Capabilities().Return(editor.Capabilities{Editable: true}).AnyTimes()
	f.mockFactory.EXPECT().Name().Return("demo").AnyTimes()
	f.mockFactory.EXPECT().DisplayLabel().Return("Demo").AnyTimes()

	f.mockState.Editors().Selector().RegisterEditor(f.mockFactory)

	t.Cleanup(f.teardown)

	return f
}

func (f *fileDetailsFixture) teardown() {
	f.ctrl.Finish()
}

func (f *fileDetailsFixture) AppCtx() *mocks_appcontext.MockAppContext {
	return f.mockAppCtx
}

func (f *fileDetailsFixture) State() *state.State {
	return f.mockState
}

func (f *fileDetailsFixture) File() *directory.File {
	return f.file
}

func (f *fileDetailsFixture) WithFile(file *directory.File) *fileDetailsFixture {
	f.file = file
	return f
}

func (f *fileDetailsFixture) LimitBytes() uint64 {
	return f.limitBytes
}

func (f *fileDetailsFixture) MockFactory() *mock_editor.MockFactory {
	return f.mockFactory
}

func (f *fileDetailsFixture) ExplorerViewModel() *mocks_viewmodel.MockExplorerViewModel {
	return f.mockExplorerVM
}

func (f *fileDetailsFixture) ConnectionViewModel() *mocks_viewmodel.MockConnectionViewModel {
	return f.mockConnVM
}

func (f *fileDetailsFixture) SettingsViewModel() *mocks_viewmodel.MockSettingsViewModel {
	return f.mockSettingsVM
}

func (f *fileDetailsFixture) EditorViewModel() *mocks_viewmodel.MockEditorViewModel {
	return f.mockEditorVM
}

func (f *fileDetailsFixture) TagsViewModel() *mocks_viewmodel.MockTagsViewModel {
	return f.mockTagsVM
}

// Deprecated: Use newFileDetailsFixture instead
func setupFileDetailsMocks(t *testing.T) *fileDetailsFixture {
	return newFileDetailsFixture(t, 20*1024)
}

// Deprecated: Use newFileDetailsFixture instead
func setupFileDetailsMocksWithLimit(t *testing.T, limitBytes uint64) *fileDetailsFixture {
	return newFileDetailsFixture(t, limitBytes)
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
		res := widget.NewFileDetails(m.AppCtx())
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		tu.AssertImageMatches(t, "images/file-details.png", c.Capture())
	})

	t.Run("should disable edit button when file is too large", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocksWithLimit(t, 512)

		// When
		res := widget.NewFileDetails(m.AppCtx())
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		tu.AssertImageMatches(t, "images/file-details-too-large.png", c.Capture())
	})

	t.Run("should enable edit button when file is too large and is not editable", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocksWithLimit(t, 512)

		nonEditableFactory := mock_editor.NewMockFactory(m.ctrl)
		nonEditableFactory.EXPECT().Capabilities().Return(editor.Capabilities{Editable: false}).AnyTimes()
		nonEditableFactory.EXPECT().Name().Return("imageviewer").AnyTimes()
		nonEditableFactory.EXPECT().DisplayLabel().Return("Image Viewer").AnyTimes()
		nonEditableFactory.EXPECT().DefaultFileRegexpPattern().Return("\\.png$").AnyTimes()

		m.State().Editors().Selector().RegisterEditor(nonEditableFactory)
		u.Skip(m.State().Editors().Selector().RegisterMapping("imageviewer", "\\.png$"))

		var file *directory.File
		tu.MakeDirectory(t, "", tu.AsRoot(),
			tu.WithFileTo("test.png", &file,
				directory.WithFileSize(fakeFileSizeLimitKB),
				directory.WithFileLastModified(lastModified)))

		// When
		res := widget.NewFileDetails(m.AppCtx())
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		tu.AssertImageMatches(t, "images/file-details-too-large-non-editable.png", c.Capture())
	})

	t.Run("should disable delete if read-only", func(t *testing.T) {
		// Given
		m := setupFileDetailsMocks(t)

		deck := m.AppCtx().State().Connection().Deck()
		conn := deck.New("Conn 1", "ak1", "sk1", "b1",
			connection_deck.WithReadOnlyOption(true)).
			Payload().(connection_deck.CreateConnectionTriggered).Connection()
		u.SkipV(deck.Select(conn.ID()))

		// When
		res := widget.NewFileDetails(m.AppCtx())
		res.Select(file)
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		tu.AssertImageMatches(t, "images/file-details-read-only.png", c.Capture())
	})
}
