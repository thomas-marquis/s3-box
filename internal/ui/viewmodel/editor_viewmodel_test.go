package viewmodel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
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
		connection_deck.WithID(fakeConnID)).Connection()

	t.Run("should open the editor and then load the file content", func(t *testing.T) {
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
		ctx := context.TODO()

		file, err := directory.NewFile("test.txt", directory.RootPath)
		require.NoError(t, err)

		// When opening the editor
		res, err := vm.Open(ctx, file)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, file, res.File)
		isLoaded, _ := res.IsLoaded.Get()
		assert.False(t, isLoaded)
		fileContent, _ := res.Content.Get()
		assert.Equal(t, "", fileContent)
		errMsg, _ := res.ErrorMsg.Get()
		assert.Equal(t, "", errMsg)

		// When the file is loaded
		fo := &directory.InMemoryFileObject{Data: []byte("Hello world!")}
		eventsChan <- directory.NewFileLoadSuccessEvent(file, fo)

		// Then
		assert.Eventually(t, func() bool {
			loaded, _ := res.IsLoaded.Get()
			return loaded
		}, 5*time.Second, 100*time.Millisecond)

		isLoaded, _ = res.IsLoaded.Get()
		assert.True(t, isLoaded)
		fileContent, _ = res.Content.Get()
		assert.Equal(t, "Hello world!", fileContent)
		errMsg, _ = res.ErrorMsg.Get()
		assert.Equal(t, "", errMsg)
	})

	t.Run("should display an error message when the file cannot be loaded", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		fyne_test.NewWindow(nil)

		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifier := mocks_notification.NewMockRepository(ctrl)

		expectedErr := errors.New("file loading failed")
		mockNotifier.EXPECT().NotifyError(gomock.Eq(expectedErr)).Times(1)

		eventsChan := make(chan event.Event)
		defer close(eventsChan)

		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(eventsChan)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).Do(func(event event.Event) {
			eventsChan <- event
		}).AnyTimes()

		vm := viewmodel.NewEditorViewModel(mockBus, mockNotifier, conn)
		ctx := context.TODO()

		file, err := directory.NewFile("test.txt", directory.RootPath)
		require.NoError(t, err)

		// When opening the editor
		res, err := vm.Open(ctx, file)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, file, res.File)
		isLoaded, _ := res.IsLoaded.Get()
		assert.False(t, isLoaded)
		fileContent, _ := res.Content.Get()
		assert.Equal(t, "", fileContent)
		errMsg, _ := res.ErrorMsg.Get()
		assert.Equal(t, "", errMsg)

		// When, file loading fails
		eventsChan <- directory.NewFileLoadFailureEvent(expectedErr, file)

		// Then
		assert.Eventually(t, func() bool {
			loaded, _ := res.IsLoaded.Get()
			return loaded
		}, 5*time.Second, 100*time.Millisecond)

		isLoaded, _ = res.IsLoaded.Get()
		assert.True(t, isLoaded)
		fileContent, _ = res.Content.Get()
		assert.Equal(t, "", fileContent)
		errMsg, _ = res.ErrorMsg.Get()
		assert.Equal(t, "file loading failed", errMsg)
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
		ctx := context.TODO()

		file, err := directory.NewFile("test.txt", directory.RootPath)
		require.NoError(t, err)

		// When
		eventsChan <- connection_deck.NewRemoveSuccessEvent(fakeDeck, conn)
		require.Eventually(t, func() bool {
			return vm.SelectedConnection() == nil
		}, 5*time.Second, 100*time.Millisecond)

		_, err = vm.Open(ctx, file)

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
		ctx := context.TODO()

		file, err := directory.NewFile("test.txt", directory.RootPath)
		require.NoError(t, err)

		file2, err := directory.NewFile("test2.txt", directory.RootPath)
		require.NoError(t, err)

		oe1, _ := vm.Open(ctx, file)
		_, err = vm.Open(ctx, file2)
		require.NoError(t, err)

		// When
		_, err = vm.Open(ctx, file)

		// Then
		assert.Equal(t, viewmodel.ErrEditorAlreadyOpened, err)
		var _ = oe1 // TODO: assert the oe1 gained the focus again (it's not yet possible with fyne...)
	})
}

func TestEditorViewModelImpl_IsOpened(t *testing.T) {
	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	conn := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID)).Connection()

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
		ctx := context.TODO()

		file1, err := directory.NewFile("test1.txt", directory.RootPath)
		require.NoError(t, err)
		file2, err := directory.NewFile("test1.txt", directory.RootPath)
		require.NoError(t, err)
		file3, err := directory.NewFile("test1.txt", directory.NewPath("mydir"))
		require.NoError(t, err)

		// When & Then
		assert.False(t, vm.IsOpened(file1))
		assert.False(t, vm.IsOpened(file2))
		assert.False(t, vm.IsOpened(file3))

		vm.Open(ctx, file1) // nolint:errcheck
		vm.Open(ctx, file2) // nolint:errcheck

		assert.True(t, vm.IsOpened(file1))
		assert.True(t, vm.IsOpened(file2))
		assert.False(t, vm.IsOpened(file3))
	})
}

func TestEditorViewModelImpl_Close(t *testing.T) {
	fakeConnID := connection_deck.NewConnectionID()
	fakeDeck := connection_deck.New()
	conn := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID)).Connection()

	t.Run("should close opened file", func(t *testing.T) {
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
		ctx := context.TODO()

		file1, err := directory.NewFile("test1.txt", directory.RootPath)
		require.NoError(t, err)
		file2, err := directory.NewFile("test1.txt", directory.RootPath)
		require.NoError(t, err)
		file3, err := directory.NewFile("test1.txt", directory.NewPath("mydir"))
		require.NoError(t, err)

		// When & Then
		oe1, _ := vm.Open(ctx, file1)
		vm.Open(ctx, file3) // nolint:errcheck

		vm.Close(oe1)

		assert.False(t, vm.IsOpened(file1))
		assert.False(t, vm.IsOpened(file2))
		assert.True(t, vm.IsOpened(file3))
	})
}

func TestEditorViewModelImpl_connectionChanged(t *testing.T) {
	fakeDeck := connection_deck.New()

	fakeConnID1 := connection_deck.NewConnectionID()
	conn1 := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID1)).Connection()

	conn1updated := fakeDeck.New("Test connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, false),
		connection_deck.WithID(fakeConnID1)).Connection()

	fakeConnID2 := connection_deck.NewConnectionID()
	conn2 := fakeDeck.New("New connection", fakeAccessKeyId, fakeSecretAccessKey, fakeBucketName,
		connection_deck.AsS3Like(fakeEndpoint, true),
		connection_deck.WithID(fakeConnID2)).Connection()

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
		eventsChan <- connection_deck.NewSelectSuccessEvent(fakeDeck, conn2)

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
		eventsChan <- connection_deck.NewUpdateSuccessEvent(fakeDeck, conn1updated)

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
		eventsChan <- connection_deck.NewRemoveSuccessEvent(fakeDeck, conn2)

		// Then
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Nil(t, vm.SelectedConnection())
		}, 5*time.Second, 100*time.Millisecond)
	})

}
