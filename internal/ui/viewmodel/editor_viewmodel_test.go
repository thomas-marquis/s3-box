package viewmodel_test

import (
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

const (
	fakeAccessKeyId     = "AZERTY"
	fakeSecretAccessKey = "dfhdh2432J4bbhjkb"
	fakeEndpoint        = "http://localhost:4566"
	fakeBucketName      = "test-bucket"
)

func TestEditorViewModelImpl_Open(t *testing.T) {
	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	conn := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID)).
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	t.Run("should open the editor and then load the file content", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		mockEditor := mock_editor.NewMockEditor(ctrl)
		edFactory := func(bus event.Bus, win fyne.Window, file *directory.File) editor.Editor {
			return mockEditor
		}

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		fo := &directory.InMemoryContent{Data: []byte("Hello world!")}

		done := make(chan struct{})
		mockEditor.EXPECT().
			OnLoaded(gomock.Eq(fo), gomock.Nil()).
			Do(func(directory.FileContent, error) {
				close(done)
			}).
			Times(1)

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)
		vm.RegisterEditorFactory(edFactory)

		var file *directory.File
		testutil.MakeDirectory(t, "",
			testutil.AsRoot(), testutil.WithConnectionId(conn.ID()),
			testutil.WithFiles("test.txt"), testutil.FileTo("test.txt", &file))

		// When opening the editor
		assert.False(t, vm.IsOpen(file))
		ed, err := vm.Open(file)

		// Then
		assert.Equal(t, ed, mockEditor)
		assert.NoError(t, err)
		assert.True(t, vm.IsOpen(file))

		// When the file is loaded
		eventsChan <- event.New(directory.LoadFileSucceeded{
			File:    file,
			Content: fo,
		})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should display an error message when the file cannot be loaded", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		mockEditor := mock_editor.NewMockEditor(ctrl)
		edFactory := func(bus event.Bus, win fyne.Window, file *directory.File) editor.Editor {
			return mockEditor
		}

		expectedErr := errors.New("file loading failed")
		mockNotifier.EXPECT().NotifyError(gomock.Eq(expectedErr)).Times(1)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		done := make(chan struct{})
		mockEditor.EXPECT().
			OnLoaded(gomock.Nil(), gomock.Eq(expectedErr)).
			Do(func(directory.FileContent, error) {
				close(done)
			}).
			Times(1)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)
		vm.RegisterEditorFactory(edFactory)

		var file *directory.File
		testutil.MakeDirectory(t, "",
			testutil.AsRoot(), testutil.WithConnectionId(conn.ID()),
			testutil.WithFiles("test.txt"), testutil.FileTo("test.txt", &file))

		// When opening the editor
		_, err := vm.Open(file)

		// Then
		assert.NoError(t, err)
		assert.True(t, vm.IsOpen(file))

		// When, file loading fails
		eventsChan <- event.New(directory.LoadFileFailed{
			Err:  expectedErr,
			File: file,
		})

		// Then
		assert.True(t, vm.IsOpen(file))
		testutil.AssertEventually(t, done)
	})

	t.Run("should return an error when no connection is selected", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)

		var file *directory.File
		testutil.MakeDirectory(t, "",
			testutil.AsRoot(), testutil.WithConnectionId(conn.ID()),
			testutil.WithFiles("test.txt"), testutil.FileTo("test.txt", &file))

		// When
		eventsChan <- event.New(connection_deck.RemoveConnectionSucceeded{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: conn},
			Deck:              fakeDeck,
		})
		require.Eventually(t, func() bool {
			return vm.SelectedConnection() == nil
		}, 5*time.Second, 100*time.Millisecond)

		_, err := vm.Open(file)

		// Then
		assert.Equal(t, viewmodel.ErrNoConnectionSelected, err)
	})

	t.Run("should focus on an already opened file", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)

		var file, file2 *directory.File
		testutil.MakeDirectory(t, "",
			testutil.AsRoot(), testutil.WithConnectionId(conn.ID()),
			testutil.WithFiles("test.txt", "test2.txt"),
			testutil.FileTo("test.txt", &file), testutil.FileTo("test2.txt", &file2))

		oe1, _ := vm.Open(file)
		_, err := vm.Open(file2)
		require.NoError(t, err)

		// When
		_, err = vm.Open(file)

		// Then
		assert.Equal(t, viewmodel.ErrEditorAlreadyOpened, err)
		var _ = oe1 // TODO: assert the oe1 gained the focus again (it's not yet possible with fyne...)
	})
}

func TestEditorViewModelImpl_IsOpen(t *testing.T) {
	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	conn := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID)).
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	t.Run("should return true when the file is opened, false otherwise", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)

		var file1, file2, file3 *directory.File
		testutil.MakeDirectory(t, "",
			testutil.AsRoot(), testutil.WithConnectionId(conn.ID()),
			testutil.WithFiles("test1.txt", "test2.txt", "test3.txt"),
			testutil.FileTo("test1.txt", &file1), testutil.FileTo("test2.txt", &file2),
			testutil.WithSubDirectory("mydir",
				testutil.WithFiles("test3.txt"),
				testutil.FileTo("test3.txt", &file3)))

		// When & Then
		assert.False(t, vm.IsOpen(file1))
		assert.False(t, vm.IsOpen(file2))
		assert.False(t, vm.IsOpen(file3))

		vm.Open(file1) // nolint:errcheck
		vm.Open(file2) // nolint:errcheck

		assert.True(t, vm.IsOpen(file1))
		assert.True(t, vm.IsOpen(file2))
		assert.False(t, vm.IsOpen(file3))
	})
}

func TestEditorViewModelImpl_Close(t *testing.T) {
	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	conn := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID)).
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	t.Run("should close opened file depending on the editor feedback", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		mockEditorFactory := func(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
			mockEditor := mock_editor.NewMockEditor(ctrl)
			if file.Name() == "test1.txt" {
				mockEditor.EXPECT().Close().Return(true).Times(1)
			} else if file.Name() == "test2.txt" {
				mockEditor.EXPECT().Close().Return(false).Times(1)
			} else {
				mockEditor.EXPECT().Close().Times(0)
			}
			return mockEditor
		}

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)
		vm.RegisterEditorFactory(mockEditorFactory)

		var file1, file2, file3 *directory.File
		testutil.MakeDirectory(t, "",
			testutil.AsRoot(), testutil.WithConnectionId(conn.ID()),
			testutil.WithFiles("test1.txt", "test2.txt", "test3.txt"),
			testutil.FileTo("test1.txt", &file1), testutil.FileTo("test2.txt", &file2),
			testutil.WithSubDirectory("mydir",
				testutil.WithFiles("test3.txt"),
				testutil.FileTo("test3.txt", &file3)))

		// When & Then
		vm.Open(file1) // nolint:errcheck
		vm.Open(file3) // nolint:errcheck
		vm.Open(file2) // nolint:errcheck

		vm.Close(file1)
		vm.Close(file2)

		assert.False(t, vm.IsOpen(file1))
		assert.True(t, vm.IsOpen(file2))
		assert.True(t, vm.IsOpen(file3))
	})
}

func TestEditorViewModelImpl_connectionChanged(t *testing.T) {
	fakeDeck := connection_deck.New()

	fakeConnID1 := connection_deck.NewConnectionID()
	conn1 := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID1)).
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	conn1updated := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID1)).
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	fakeConnID2 := connection_deck.NewConnectionID()
	conn2 := fakeDeck.New("New connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, true),
		connection_deck.WithID(fakeConnID2)).
		Payload().(connection_deck.CreateConnectionTriggered).Connection()

	t.Run("should set the new connection when selected", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn1)

		// When
		eventsChan <- event.New(connection_deck.SelectConnectionSucceeded{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: conn2},
			Deck:              fakeDeck,
		})

		// Then
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.NotNil(t, vm.SelectedConnection())
			assert.Equal(t, conn2, vm.SelectedConnection())
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("should set the new connection when updated", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn1)

		// When
		eventsChan <- event.New(connection_deck.UpdateConnectionSucceeded{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: conn1updated},
			Deck:              fakeDeck,
		})

		// Then
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.NotNil(t, vm.SelectedConnection())
			assert.Equal(t, conn1updated, vm.SelectedConnection())
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("should reset the connection when deleted", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn1)

		// When
		eventsChan <- event.New(connection_deck.RemoveConnectionSucceeded{
			ConnectionPayload: connection_deck.ConnectionPayload{Conn: conn2},
			Deck:              fakeDeck,
		})

		// Then
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Nil(t, vm.SelectedConnection())
		}, 5*time.Second, 100*time.Millisecond)
	})

}
