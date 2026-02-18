package viewmodel

import (
	"context"
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

var (
	ErrEditorAlreadyOpened = fmt.Errorf("editor already opened")
)

type OpenedEditor struct {
	Window fyne.Window
	File   *directory.File

	Content  binding.String
	IsLoaded binding.Bool
	ErrorMsg binding.String

	OnSave func(fileContent string) error
}

type EditorViewModel interface {
	ViewModel

	SelectedConnection() *connection_deck.Connection

	// Open opens the given file in a new editor window.
	// Returns an ErrAlreadyOpened error if the file is already opened.
	Open(ctx context.Context, file *directory.File) (*OpenedEditor, error)

	IsOpened(file *directory.File) bool
	Close(editor *OpenedEditor)
}

type editorViewModelImpl struct {
	baseViewModel

	openedEditors      map[string]*OpenedEditor
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
		openedEditors:      make(map[string]*OpenedEditor),
		bus:                bus,
		notifier:           notifier,
		selectedConnection: initialConnection,
	}

	bus.Subscribe().
		On(event.Is(directory.FileLoadEventType.AsSuccess()), vm.handleFileLoadingSuccess).
		On(event.Is(directory.FileLoadEventType.AsFailure()), vm.handleFileLoadingFailure).
		On(event.IsOneOf(connection_deck.SelectEventType.AsSuccess(),
			connection_deck.UpdateEventType.AsSuccess(),
			connection_deck.RemoveEventType.AsSuccess()), vm.handleConnectionChanged).
		ListenWithWorkers(1)

	return vm
}

func (v *editorViewModelImpl) SelectedConnection() *connection_deck.Connection {
	return v.selectedConnection
}

func (v *editorViewModelImpl) Open(ctx context.Context, file *directory.File) (*OpenedEditor, error) {
	if v.selectedConnection == nil {
		return nil, ErrNoConnectionSelected
	}

	if oe, ok := v.openedEditors[file.FullPath()]; ok {
		oe.Window.RequestFocus()
		return nil, ErrEditorAlreadyOpened
	}

	oe := &OpenedEditor{
		Window:   fyne.CurrentApp().NewWindow(file.Name().String()),
		File:     file,
		OnSave:   func(string) error { return nil },
		Content:  binding.NewString(),
		IsLoaded: binding.NewBool(),
		ErrorMsg: binding.NewString(),
	}
	v.openedEditors[file.FullPath()] = oe

	v.bus.Publish(file.Load(v.selectedConnection.ID(), event.WithContext(ctx)))

	return oe, nil
}

func (v *editorViewModelImpl) handleFileLoadingSuccess(evt event.Event) {
	e := evt.(directory.FileLoadSuccessEvent)
	content := e.Content
	oe, ok := v.openedEditors[e.File().FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}

	if _, err := content.Seek(0, io.SeekStart); err != nil {
		v.notifier.NotifyError(err)
		oe.IsLoaded.Set(true)
		oe.ErrorMsg.Set(err.Error())
		return
	}

	oe.OnSave = func(newContent string) error {
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := fmt.Fprint(content, newContent); err != nil {
			return err
		}
		return nil
	}

	contentVal, err := io.ReadAll(content)
	if err != nil {
		v.notifier.NotifyError(err)
		oe.IsLoaded.Set(true)
		oe.ErrorMsg.Set(err.Error())
		return
	}

	oe.Content.Set(string(contentVal))
	oe.IsLoaded.Set(true)
}

func (v *editorViewModelImpl) handleFileLoadingFailure(evt event.Event) {
	e := evt.(directory.FileLoadFailureEvent)
	v.notifier.NotifyError(e.Error())
	oe, ok := v.openedEditors[e.File().FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}
	oe.ErrorMsg.Set(e.Error().Error())
	oe.IsLoaded.Set(true)
}

func (v *editorViewModelImpl) IsOpened(file *directory.File) bool {
	_, ok := v.openedEditors[file.FullPath()]
	return ok
}

func (v *editorViewModelImpl) Close(editor *OpenedEditor) {
	if _, ok := v.openedEditors[editor.File.FullPath()]; ok {
		delete(v.openedEditors, editor.File.FullPath())
	}
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
			v.Close(oe)
		}
		v.selectedConnection = conn
	}
}
