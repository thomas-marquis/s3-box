package viewmodel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/eventest"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	mocks_fyne "github.com/thomas-marquis/s3-box/mocks/fyne"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

const (
	fakeAccessKeyId     = "AZERTY"
	fakeSecretAccessKey = "dfhdh2432J4bbhjkb"
	fakeEndpoint        = "http://localhost:4566"
	fakeBucketName      = "test-bucket"
)

type editorVMFixture struct {
	ctrl      *gomock.Controller
	ctx       context.Context
	cancel    context.CancelFunc
	app       fyne.App
	conn      *connection_deck.Connection
	deck      *connection_deck.Deck
	bus       event.Bus
	harvester *event.HarvesterNotifier
	notifier  *mocks_notification.MockRepository
	t         *testing.T
	ready     chan struct{}
	state     *state.State
}

func setupEditorVM(t *testing.T) *editorVMFixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	f := &editorVMFixture{
		ctrl:  ctrl,
		app:   fyne_test.NewApp(),
		t:     t,
		ready: make(chan struct{}),
		state: state.New(),
	}

	f.ctx, f.cancel = context.WithCancel(context.Background())

	t.Cleanup(f.tearDown)

	return f
}

func (f *editorVMFixture) Deck() *connection_deck.Deck {
	f.t.Helper()
	if f.deck == nil {
		f.deck = connection_deck.New()
	}
	return f.deck
}
func (f *editorVMFixture) State() *state.State {
	return f.state
}

func (f *editorVMFixture) Connection() *connection_deck.Connection {
	f.t.Helper()
	if f.conn == nil {
		fakeConnID := connection_deck.NewConnectionID()
		f.conn = f.Deck().New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
			connection_deck.AsS3Like(fakeEndpoint, false),
			connection_deck.WithID(fakeConnID)).
			Payload().(connection_deck.CreateConnectionTriggered).Connection()
	}
	u.Skip(f.state.Connection().Selected().Set(f.conn))
	return f.conn
}

func (f *editorVMFixture) App() fyne.App {
	f.t.Helper()
	return f.app
}

func (f *editorVMFixture) Harvester() *event.HarvesterNotifier {
	f.t.Helper()
	if f.harvester == nil {
		f.harvester = &event.HarvesterNotifier{}
	}
	return f.harvester
}

func (f *editorVMFixture) Bus() event.Bus {
	f.t.Helper()
	if f.bus == nil {
		f.bus = inmemory.NewBus(f.ctx,
			inmemory.WithNotifier(f.Harvester()),
			inmemory.WithReadiness(f.ready))
	}
	return f.bus
}

func (f *editorVMFixture) BusReady() chan struct{} {
	f.t.Helper()
	return f.ready
}

func (f *editorVMFixture) Instance() viewmodel.EditorViewModel {
	f.t.Helper()
	vm := viewmodel.NewEditorViewModel(f.ctx, f.Bus(), f.Notifier(), f.State())
	<-f.BusReady()
	return vm
}

func (f *editorVMFixture) NewMockEditor() *mock_editor.MockEditor {
	f.t.Helper()
	mock := mock_editor.NewMockEditor(f.ctrl)
	mockWindow := mocks_fyne.NewMockWindow(f.ctrl)
	mockWindow.EXPECT().Close().AnyTimes()
	mock.EXPECT().Window().Return(mockWindow).AnyTimes()
	return mock
}

func (f *editorVMFixture) Notifier() *mocks_notification.MockRepository {
	f.t.Helper()
	if f.notifier == nil {
		f.notifier = mocks_notification.NewMockRepository(f.ctrl)
	}
	return f.notifier
}

func (f *editorVMFixture) tearDown() {
	f.cancel()
}

func TestEditorViewModelImpl_Open(t *testing.T) {
	t.Run("should open the editor and then load the file content", func(t *testing.T) {
		// Given
		fxt := setupEditorVM(t)

		mockEditor := fxt.NewMockEditor()

		edFactory := func(bus event.Bus, win fyne.Window, file *directory.File) editor.Editor {
			assert.Equal(t, "test.txt", win.Title())
			return mockEditor
		}

		fo := &directory.InMemoryContent{Data: []byte("Hello world!")}

		vm := fxt.Instance()
		vm.RegisterEditorFactory("text", edFactory)

		var file *directory.File
		tu.MakeDirectory(t, "",
			tu.AsRoot(), tu.WithConnectionId(fxt.Connection().ID()),
			tu.WithFileTo("test.txt", &file))

		// When opening the editor
		require.False(t, vm.IsOpen(file))
		ed, err := vm.Open(file)

		// Then
		assert.Equal(t, ed, mockEditor)
		assert.NoError(t, err)
		assert.True(t, vm.IsOpen(file))

		// When the file is loaded
		fxt.Bus().Publish(event.New(directory.LoadFileSucceeded{
			File:    file,
			Content: fo,
		}))

		// Then
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			assert.True(ct, vm.IsOpen(file))
			if assert.Len(ct, fxt.Harvester().Events(), 3) {
				assert.Equal(ct, directory.LoadFileTriggeredType, fxt.Harvester().Events()[0].Type())
				assert.Equal(ct, directory.LoadFileSucceededType, fxt.Harvester().Events()[1].Type())
				assert.Equal(ct, editor.LoadedType, fxt.Harvester().Events()[2].Type())
			}
			assert.Len(ct, fxt.App().Driver().AllWindows(), 2)
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should emit a load failed event the file cannot be loaded", func(t *testing.T) {
		// Given
		fxt := setupEditorVM(t)

		edFactory := func(bus event.Bus, win fyne.Window, file *directory.File) editor.Editor {
			return fxt.NewMockEditor()
		}

		expectedErr := errors.New("file loading failed")
		fxt.Notifier().EXPECT().NotifyError(gomock.Eq(expectedErr)).Times(1)

		vm := fxt.Instance()
		vm.RegisterEditorFactory("text", edFactory)

		var file *directory.File
		tu.MakeDirectory(t, "",
			tu.AsRoot(), tu.WithConnectionId(fxt.Connection().ID()),
			tu.WithFileTo("test.txt", &file))

		// When opening the editor
		_, err := vm.Open(file)

		// Then
		assert.NoError(t, err)
		assert.True(t, vm.IsOpen(file))

		// When, file loading fails
		fxt.Bus().Publish(event.New(directory.LoadFileFailed{
			Err:  expectedErr,
			File: file,
		}))

		// Then
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			assert.True(ct, vm.IsOpen(file))
			if assert.Len(ct, fxt.Harvester().Events(), 3) {
				assert.Equal(ct, editor.LoadFailedType, fxt.Harvester().Events()[2].Type())
				assert.Equal(ct, expectedErr.Error(), fxt.Harvester().Events()[2].Payload().(editor.LoadFailed).Err.Error())
			}
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should focus on an already opened file", func(t *testing.T) {
		// Given
		fxt := setupEditorVM(t)
		vm := fxt.Instance()

		var file, file2 *directory.File
		tu.MakeDirectory(t, "",
			tu.AsRoot(), tu.WithConnectionId(fxt.Connection().ID()),
			tu.WithFileTo("test.txt", &file), tu.WithFileTo("test2.txt", &file2))

		oe1, _ := vm.Open(file)
		_, err := vm.Open(file2)
		require.NoError(t, err)

		// When
		_, err = vm.Open(file)

		// Then
		assert.Equal(t, viewmodel.ErrEditorAlreadyOpened, err)
		windows := fxt.App().Driver().AllWindows()
		assert.Len(t, windows, 3)
		assert.NotNil(t, oe1)
	})
}

func TestEditorViewModelImpl_IsOpen(t *testing.T) {
	t.Run("should return true when the file is opened, false otherwise", func(t *testing.T) {
		// Given
		fxt := setupEditorVM(t)
		vm := fxt.Instance()

		var file1, file2, file3 *directory.File
		tu.MakeDirectory(t, "",
			tu.AsRoot(), tu.WithConnectionId(fxt.Connection().ID()),
			tu.WithFile("test3.txt"),
			tu.WithFileTo("test1.txt", &file1), tu.WithFileTo("test2.txt", &file2),
			tu.WithSubDirectory("mydir",
				tu.WithFileTo("test3.txt", &file3)))

		// When & Then
		assert.False(t, vm.IsOpen(file1))
		assert.False(t, vm.IsOpen(file2))
		assert.False(t, vm.IsOpen(file3))

		require.NoError(t, u.SkipE(vm.Open(file1)))
		require.NoError(t, u.SkipE(vm.Open(file2)))

		assert.True(t, vm.IsOpen(file1))
		assert.True(t, vm.IsOpen(file2))
		assert.False(t, vm.IsOpen(file3))
	})
}

func TestEditorViewModelImpl(t *testing.T) {
	t.Run("should request to close all opened editors when selected connection changed", func(t *testing.T) {
		// Given
		fxt := setupEditorVM(t)
		vm := fxt.Instance()

		var file1, file2, file3 *directory.File
		tu.MakeDirectory(t, "", tu.AsRoot(),
			tu.WithConnectionId(fxt.Connection().ID()),
			tu.WithFileTo("test1.txt", &file1),
			tu.WithFileTo("test2.txt", &file2),
			tu.WithFileTo("test3.txt", &file3))

		require.NoError(t, u.SkipE(vm.Open(file1)))
		require.NoError(t, u.SkipE(vm.Open(file2)))
		require.NoError(t, u.SkipE(vm.Open(file3)))

		require.True(t, vm.IsOpen(file1))
		require.True(t, vm.IsOpen(file2))
		require.True(t, vm.IsOpen(file3))

		// When
		u.Skip(fxt.state.Connection().Selected().Set(nil))

		// Then
		assert.True(t, vm.IsOpen(file1))
		assert.True(t, vm.IsOpen(file2))
		assert.True(t, vm.IsOpen(file3))

		evts := fxt.Harvester().Events()
		assert.GreaterOrEqual(t, len(evts), 3)
		closReqEvts := evts[len(evts)-3:]
		eventest.AssertIsType(t, closReqEvts[0], editor.CloseRequestedType)
		eventest.AssertIsType(t, closReqEvts[1], editor.CloseRequestedType)
		eventest.AssertIsType(t, closReqEvts[2], editor.CloseRequestedType)
	})
}
