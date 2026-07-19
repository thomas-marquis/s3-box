package viewmodel

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"
)

var (
	ErrEditorAlreadyOpened = fmt.Errorf("editor already opened")
)

type EditorViewModel interface {
	ViewModel

	SelectedConnection() *connection_deck.Connection

	RegisterEditorFactory(name string, initializer editor.Initializer)

	// Open opens the given file in a new editor window.
	// Returns an ErrAlreadyOpened error if the file is already opened.
	Open(file *directory.File) (editor.Editor, error)

	IsOpen(file *directory.File) bool
	Close(file *directory.File)
}

type editorViewModelImpl struct {
	baseViewModel
	mu sync.Mutex

	openedEditors      map[string]editor.Editor
	loadedContents     map[string]directory.FileContent
	selectedConnection *connection_deck.Connection
	editorFactories    map[string]editor.Initializer

	bus      event.Bus
	notifier notification.Repository
}

func NewEditorViewModel(
	bus event.Bus,
	notifier notification.Repository,
	initialConnection *connection_deck.Connection,
) EditorViewModel {
	vm := &editorViewModelImpl{
		openedEditors:      make(map[string]editor.Editor),
		loadedContents:     make(map[string]directory.FileContent),
		bus:                bus,
		notifier:           notifier,
		selectedConnection: initialConnection,
		editorFactories: map[string]editor.Initializer{
			"text": texteditor.New,
			"csv":  csveditor.New,
		},
	}

	bus.Subscribe().
		On(event.Is(directory.LoadFileSucceededType), vm.handleFileLoadingSuccess).
		On(event.Is(directory.LoadFileFailedType), vm.handleFileLoadingFailure).
		On(event.IsOneOf(
			connection_deck.SelectConnectionSucceededType,
			connection_deck.UpdateConnectionSucceededType,
			connection_deck.RemoveConnectionSucceededType,
		), vm.handleConnectionChanged).
		On(event.Is(editor.SaveTriggeredType), vm.handleFileSave).
		On(event.Is(editor.SaveSucceededType), vm.handleFileSaveSuccess).
		On(event.Is(editor.SaveFailedType), vm.handleFileSaveFailure).
		ListenNonBlocking()

	return vm
}

func (v *editorViewModelImpl) SelectedConnection() *connection_deck.Connection {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.selectedConnection
}

func (v *editorViewModelImpl) RegisterEditorFactory(name string, initializer editor.Initializer) {
	v.editorFactories[name] = initializer
}

func (v *editorViewModelImpl) Open(file *directory.File) (editor.Editor, error) {
	if v.selectedConnection == nil {
		return nil, ErrNoConnectionSelected
	}

	if e, ok := v.openedEditors[file.FullPath()]; ok {
		e.Window().RequestFocus()
		return e, ErrEditorAlreadyOpened
	}

	newWin := fyne.CurrentApp().NewWindow(file.Name().String())

	var e editor.Editor
	if strings.HasSuffix(file.Name().String(), ".csv") {
		e = v.editorFactories["csv"](v.bus, newWin, file)
	} else {
		e = v.editorFactories["text"](v.bus, newWin, file)
	}

	v.openedEditors[file.FullPath()] = e

	ctx, cancel := context.WithCancel(context.Background())

	if _, ok := e.(editor.Closable); ok {
		newWin.SetCloseIntercept(func() {
			v.closeEditor(file, cancel)
		})
	} else {
		newWin.SetOnClosed(func() {
			cancel()
			v.unregisterEditor(file)
		})
	}

	v.bus.Publish(file.Load(v.selectedConnection.ID(), event.WithContext(ctx)))

	return e, nil
}

func (v *editorViewModelImpl) handleFileLoadingSuccess(evt event.Event) {
	pl := evt.Payload().(directory.LoadFileSucceeded)

	e, ok := v.openedEditors[pl.File.FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}

	if _, err := pl.Content.Seek(0, io.SeekStart); err != nil {
		v.notifier.NotifyError(err)
		fyne.Do(func() {
			e.OnLoaded(nil, err)
		})
		return
	}

	v.mu.Lock()
	v.loadedContents[pl.File.FullPath()] = pl.Content
	v.mu.Unlock()

	fyne.Do(func() {
		e.OnLoaded(pl.Content, nil)
	})
}

func (v *editorViewModelImpl) handleFileLoadingFailure(evt event.Event) {
	pl := evt.Payload().(directory.LoadFileFailed)
	v.notifier.NotifyError(pl.Err)

	e, ok := v.openedEditors[pl.File.FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}
	fyne.Do(func() {
		e.OnLoaded(nil, pl.Err)
	})
}

func (v *editorViewModelImpl) IsOpen(file *directory.File) bool {
	_, ok := v.openedEditors[file.FullPath()]
	return ok
}

func (v *editorViewModelImpl) Close(file *directory.File) {
	v.closeEditor(file, nil)
}

func (v *editorViewModelImpl) closeEditor(file *directory.File, onClose func()) {
	ed, ok := v.openedEditors[file.FullPath()]
	if !ok {
		return
	}

	if closable, ok := ed.(editor.Closable); ok {
		closable.BeforeClose(func(ready bool) {
			if ready {
				if onClose != nil {
					onClose()
				}
				ed.Window().Close()
				v.unregisterEditor(file)
			} else {
				ed.Window().RequestFocus()
			}
		})
	}
}

func (v *editorViewModelImpl) unregisterEditor(file *directory.File) {
	path := file.FullPath()
	if !v.IsOpen(file) {
		return
	}
	delete(v.openedEditors, path)
	delete(v.loadedContents, path)
}

func (v *editorViewModelImpl) handleFileSave(e event.Event) {
	pl := e.Payload().(editor.SaveTriggered)

	if _, isOpen := v.openedEditors[pl.File.FullPath()]; !isOpen {
		// The editor has been closed before in the meantime (unlikely). But it's okay
		return
	}

	content, isLoaded := v.loadedContents[pl.File.FullPath()]
	if !isLoaded {
		v.bus.Publish(e.NewFollowup(editor.SaveFailed{
			File: pl.File,
			Err:  fmt.Errorf("editor loading is not finished yet"),
		}))
		return
	}

	terminated := make(chan struct{})
	go func() {
		defer close(terminated)

		if _, err := content.Seek(0, io.SeekStart); err != nil {
			v.bus.Publish(e.NewFollowup(editor.SaveFailed{
				Err:  err,
				File: pl.File,
			}))
			return
		}
		if _, err := fmt.Fprint(content, pl.Content); err != nil {
			v.bus.Publish(e.NewFollowup(editor.SaveFailed{
				Err:  err,
				File: pl.File,
			}))
			return
		}
		pl.File.SetSizeBytes(uint64(len(pl.Content)))
		v.bus.Publish(e.NewFollowup(editor.SaveSucceeded(pl)))
	}()

	select {
	case <-e.Context().Done():
		content.Cancel()
	case <-terminated:
	}
}

func (v *editorViewModelImpl) handleFileSaveSuccess(evt event.Event) {
	pl := evt.Payload().(editor.SaveSucceeded)
	ed, found := v.openedEditors[pl.File.FullPath()]
	if !found {
		// the editor was closed before the save succeeded, and that's okay
		return
	}

	fyne.Do(func() {
		ed.OnSaved(pl.Content, nil)
	})

}

func (v *editorViewModelImpl) handleFileSaveFailure(evt event.Event) {
	pl := evt.Payload().(editor.SaveFailed)
	v.notifier.NotifyError(pl.Err)

	ed, found := v.openedEditors[pl.File.FullPath()]
	if !found {
		// the editor was closed before the save succeeded, and that's okay
		return
	}

	fyne.Do(func() {
		ed.OnSaved("", pl.Err)
	})
}

func (v *editorViewModelImpl) handleConnectionChanged(evt event.Event) {
	var hasChanged bool
	var conn *connection_deck.Connection
	if _, ok := evt.Payload().(connection_deck.RemoveConnectionSucceeded); ok {
		hasChanged = true
	} else {
		pl, ok := evt.Payload().(connection_deck.SelectConnectionSucceeded)
		if ok {
			conn = pl.Connection()
		} else {
			pl := evt.Payload().(connection_deck.UpdateConnectionSucceeded)
			conn = pl.Connection()
			if conn.ID() != v.selectedConnection.ID() {
				return
			}
		}

		hasChanged = (v.selectedConnection == nil && conn != nil) ||
			(v.selectedConnection != nil && conn == nil) ||
			(v.selectedConnection != nil && !v.selectedConnection.Is(conn))
	}

	if hasChanged {
		v.mu.Lock()
		for _, oe := range v.openedEditors {
			v.Close(oe.File()) // TODO: move this when the connection change is triggered and warn the user for unsaved changes before closing the editors
		}
		v.selectedConnection = conn
		v.mu.Unlock()
	}
}
