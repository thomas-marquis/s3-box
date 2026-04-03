package viewmodel

import (
	"context"
	"fmt"
	"io"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/fileeditor"
)

var (
	ErrEditorAlreadyOpened = fmt.Errorf("editor already opened")
)

type EditorViewModel interface {
	ViewModel

	SelectedConnection() *connection_deck.Connection

	// Open opens the given file in a new editor window.
	// Returns an ErrAlreadyOpened error if the file is already opened.
	Open(ctx context.Context, file *directory.File) (*fileeditor.State, error)

	IsOpened(file *directory.File) bool
	Close(file *directory.File)
}

type openedEditor struct {
	state   *fileeditor.State
	content directory.FileContent
}

type editorViewModelImpl struct {
	baseViewModel
	sync.Mutex

	openedEditors      map[string]*openedEditor
	selectedConnection *connection_deck.Connection

	bus      event.Bus
	notifier notification.Repository
}

func NewEditorViewModel(
	bus event.Bus,
	notifier notification.Repository,
	initialConnection *connection_deck.Connection,
) EditorViewModel {
	vm := &editorViewModelImpl{
		openedEditors:      make(map[string]*openedEditor),
		bus:                bus,
		notifier:           notifier,
		selectedConnection: initialConnection,
	}

	bus.Subscribe().
		On(event.Is(directory.FileLoadEventType.AsSuccess()), vm.handleFileLoadingSuccess).
		On(event.Is(directory.FileLoadEventType.AsFailure()), vm.handleFileLoadingFailure).
		On(event.IsOneOf(
			connection_deck.SelectEventType.AsSuccess(),
			connection_deck.UpdateEventType.AsSuccess(),
			connection_deck.RemoveEventType.AsSuccess(),
		), vm.handleConnectionChanged).
		On(event.Is(fileeditor.SaveEventType), vm.handleFileSave).
		On(event.Is(fileeditor.SaveEventType.AsFailure()), vm.handleFileSaveFailure).
		ListenNonBlocking()

	return vm
}

func (v *editorViewModelImpl) SelectedConnection() *connection_deck.Connection {
	v.Lock()
	defer v.Unlock()
	return v.selectedConnection
}

func (v *editorViewModelImpl) Open(ctx context.Context, file *directory.File) (*fileeditor.State, error) {
	if v.selectedConnection == nil {
		return nil, ErrNoConnectionSelected
	}

	if oe, ok := v.openedEditors[file.FullPath()]; ok {
		oe.state.Window.RequestFocus()
		return nil, ErrEditorAlreadyOpened
	}

	es := &fileeditor.State{
		Window:   fyne.CurrentApp().NewWindow(file.Name().String()),
		File:     file,
		Content:  binding.NewString(),
		IsLoaded: binding.NewBool(),
		ErrorMsg: binding.NewString(),
		Bus:      v.bus,
	}
	v.openedEditors[file.FullPath()] = &openedEditor{
		state: es,
	}

	v.bus.Publish(file.Load(v.selectedConnection.ID(), event.WithContext(ctx)))

	return es, nil
}

func (v *editorViewModelImpl) handleFileLoadingSuccess(evt event.Event) {
	e := evt.(directory.FileLoadSuccessEvent)
	content := e.Content
	oe, ok := v.openedEditors[e.File.FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}

	oe.content = content

	if _, err := content.Seek(0, io.SeekStart); err != nil {
		v.notifier.NotifyError(err)
		oe.state.IsLoaded.Set(true)        // nolint:errcheck
		oe.state.ErrorMsg.Set(err.Error()) // nolint:errcheck
		return
	}

	contentVal, err := io.ReadAll(content)
	if err != nil {
		v.notifier.NotifyError(err)
		oe.state.IsLoaded.Set(true)        // nolint:errcheck
		oe.state.ErrorMsg.Set(err.Error()) // nolint:errcheck
		return
	}

	oe.state.Content.Set(string(contentVal)) // nolint:errcheck
	oe.state.IsLoaded.Set(true)              // nolint:errcheck
}

func (v *editorViewModelImpl) handleFileLoadingFailure(evt event.Event) {
	e := evt.(directory.FileLoadFailureEvent)
	v.notifier.NotifyError(e.Error())
	oe, ok := v.openedEditors[e.File.FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}
	oe.state.ErrorMsg.Set(e.Error().Error()) // nolint:errcheck
	oe.state.IsLoaded.Set(true)              // nolint:errcheck
}

func (v *editorViewModelImpl) IsOpened(file *directory.File) bool {
	_, ok := v.openedEditors[file.FullPath()]
	return ok
}

func (v *editorViewModelImpl) Close(file *directory.File) {
	delete(v.openedEditors, file.FullPath())
}

func (v *editorViewModelImpl) handleFileSave(e event.Event) {
	evt := e.(fileeditor.SaveEvent)
	oe, ok := v.openedEditors[evt.File.FullPath()]
	if !ok {
		// The editor has been closed before in the meantime (unlikely). But it's okay
		return
	}

	terminated := make(chan struct{})

	go func() {
		defer close(terminated)
		if _, err := oe.content.Seek(0, io.SeekStart); err != nil {
			v.bus.Publish(fileeditor.NewSaveFailureEvent(evt.File, err))
			return
		}
		if _, err := fmt.Fprint(oe.content, evt.Content); err != nil {
			v.bus.Publish(fileeditor.NewSaveFailureEvent(evt.File, err))
			return
		}
		v.bus.Publish(fileeditor.NewSaveSuccessEvent(evt.File, evt.Content))
	}()

	select {
	case <-evt.Context().Done():
		oe.content.Cancel()
	case <-terminated:
	}
}

func (v *editorViewModelImpl) handleFileSaveFailure(evt event.Event) {
	e := evt.(fileeditor.SaveFailureEvent)
	v.notifier.NotifyError(e.Error())
}

func (v *editorViewModelImpl) handleConnectionChanged(evt event.Event) {
	var hasChanged bool
	var conn *connection_deck.Connection
	if _, ok := evt.(connection_deck.RemoveSuccessEvent); ok {
		hasChanged = true
	} else {
		e, ok := evt.(connection_deck.SelectSuccessEvent)
		if ok {
			conn = e.Connection()
		} else {
			e := evt.(connection_deck.UpdateSuccessEvent)
			conn = e.Connection()
			if conn.ID() != v.selectedConnection.ID() {
				return
			}
		}

		hasChanged = (v.selectedConnection == nil && conn != nil) ||
			(v.selectedConnection != nil && conn == nil) ||
			(v.selectedConnection != nil && !v.selectedConnection.Is(conn))
	}

	if hasChanged {
		for _, oe := range v.openedEditors {
			v.Close(oe.state.File)
		}
		v.Lock()
		v.selectedConnection = conn
		v.Unlock()
	}
}
