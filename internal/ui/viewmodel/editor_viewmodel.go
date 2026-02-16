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

	go vm.listen()

	return vm
}

func (vm *editorViewModelImpl) Open(ctx context.Context, file *directory.File) (*OpenedEditor, error) {
	if oe, ok := vm.openedEditors[file.FullPath()]; ok {
		oe.Window.RequestFocus()
		return nil, ErrEditorAlreadyOpened
	}

	oe := &OpenedEditor{
		Window:   fyne.CurrentApp().NewWindow(file.Name().String()), // TODO: use the dynamically truncated display name here
		File:     file,
		OnSave:   func(string) error { return nil },
		Content:  binding.NewString(),
		IsLoaded: binding.NewBool(),
		ErrorMsg: binding.NewString(),
	}
	vm.openedEditors[file.FullPath()] = oe

	vm.bus.Publish(file.Load(vm.selectedConnection.ID(), event.WithContext(ctx)))

	return oe, nil
}

func (vm *editorViewModelImpl) IsOpened(file *directory.File) bool {
	_, ok := vm.openedEditors[file.FullPath()]
	return ok
}

func (vm *editorViewModelImpl) Close(editor *OpenedEditor) {
	if _, ok := vm.openedEditors[editor.File.FullPath()]; ok {
		delete(vm.openedEditors, editor.File.FullPath())
	}
}

func (vm *editorViewModelImpl) listen() {
	events := vm.bus.Subscribe(
		connection_deck.SelectEventType.AsSuccess(),
		connection_deck.UpdateEventType.AsSuccess(),
		connection_deck.RemoveEventType.AsSuccess(),
		directory.FileLoadEventType.AsFailure(),
		directory.FileLoadEventType.AsSuccess(),
	)

	for evt := range events {
		switch evt.Type() {
		case connection_deck.SelectEventType.AsSuccess(), connection_deck.UpdateEventType.AsSuccess():
			var conn *connection_deck.Connection
			e, ok := evt.(connection_deck.SelectSuccessEvent)
			if ok {
				conn = e.Connection()
			} else {
				e := evt.(connection_deck.UpdateSuccessEvent)
				conn = e.Connection()
				if conn.ID() != vm.selectedConnection.ID() {
					continue
				}
			}
			hasChanged := (vm.selectedConnection == nil && conn != nil) ||
				(vm.selectedConnection != nil && conn == nil) ||
				(vm.selectedConnection != nil && !vm.selectedConnection.Is(conn))
			if hasChanged {
				for _, oe := range vm.openedEditors {
					vm.Close(oe)
				}
				vm.selectedConnection = conn
			}

		case connection_deck.RemoveEventType.AsSuccess():
			e := evt.(connection_deck.RemoveSuccessEvent)
			conn := e.Connection()
			if vm.selectedConnection != nil && vm.selectedConnection.Is(conn) {
				for _, oe := range vm.openedEditors {
					vm.Close(oe)
				}
				vm.selectedConnection = nil
			}

		case directory.FileLoadEventType.AsSuccess():
			e := evt.(directory.FileLoadSuccessEvent)
			content := e.Content
			oe, ok := vm.openedEditors[e.File().FullPath()]
			if !ok {
				// The editor has been closed before the file was loaded. And it's okay
				continue
			}

			if _, err := content.Seek(0, io.SeekStart); err != nil {
				vm.notifier.NotifyError(err)
				oe.IsLoaded.Set(true)
				oe.ErrorMsg.Set(err.Error())
				continue
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
				vm.notifier.NotifyError(err)
				oe.IsLoaded.Set(true)
				oe.ErrorMsg.Set(err.Error())
				return
			}

			oe.Content.Set(string(contentVal))
			oe.IsLoaded.Set(true)

		case directory.FileLoadEventType.AsFailure():
			e := evt.(directory.FileLoadFailureEvent)
			vm.notifier.NotifyError(e.Error())
			oe, ok := vm.openedEditors[e.File().FullPath()]
			if !ok {
				// The editor has been closed before the file was loaded. And it's okay
				continue
			}
			oe.IsLoaded.Set(true)
			oe.ErrorMsg.Set(e.Error().Error())

		}
	}
}
