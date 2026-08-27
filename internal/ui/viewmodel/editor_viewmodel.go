package viewmodel

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"
)

var (
	ErrEditorAlreadyOpened = fmt.Errorf("editor already opened")
)

type EditorViewModel interface {
	ViewModel

	RegisterEditorFactory(name string, initializer editor.Initializer)

	// Open opens the given file in a new editor window.
	// Returns an ErrAlreadyOpened error if the file is already opened.
	Open(file *directory.File) (editor.Editor, error)

	IsOpen(file *directory.File) bool
}

type editorViewModelImpl struct {
	baseViewModel
	mu sync.Mutex

	openedEditors   map[string]editor.Editor
	loadedContents  map[string]directory.FileContent
	editorFactories map[string]editor.Initializer
	state           *state.State
	bus             event.Bus
	notifier        notification.Repository
}

func NewEditorViewModel(
	ctx context.Context,
	bus event.Bus,
	notifier notification.Repository,
	st *state.State,
) EditorViewModel {
	vm := &editorViewModelImpl{
		openedEditors:  make(map[string]editor.Editor),
		loadedContents: make(map[string]directory.FileContent),
		bus:            bus,
		notifier:       notifier,
		editorFactories: map[string]editor.Initializer{
			"text": texteditor.New,
			"csv":  csveditor.New,
		},
		state: st,
	}

	st.Connection().Selected().AddListener(binding.NewDataListener(vm.closeAll))

	bus.Subscribe().
		On(event.Is(directory.LoadFileSucceededType), vm.handleFileLoadingSuccess).
		On(event.Is(directory.LoadFileFailedType), vm.handleFileLoadingFailure).
		On(event.Is(editor.CloseConfirmedType), vm.handleEditorCloseConfirmed).
		On(event.Is(editor.CloseCanceledType), vm.handleEditorCloseCanceled).
		ListenNonBlocking()

	go func() {
		<-ctx.Done()
		vm.mu.Lock()
		vm.forceCloseAll() // TODO: implement a back signal to prevent from closing unsaved editors
		vm.mu.Unlock()
	}()

	return vm
}

func (v *editorViewModelImpl) RegisterEditorFactory(name string, initializer editor.Initializer) {
	v.editorFactories[name] = initializer
}

func (v *editorViewModelImpl) Open(file *directory.File) (editor.Editor, error) {
	if e, ok := v.openedEditors[file.FullPath()]; ok {
		fyne.Do(e.Window().RequestFocus)
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

	newWin.SetCloseIntercept(func() {
		v.requestClose(e, cancel)
	})

	v.bus.Publish(file.Load(u.SkipV(v.state.Connection().Selected().Get()).ID(), event.WithContext(ctx)))

	return e, nil
}

func (v *editorViewModelImpl) requestClose(e editor.Editor, cancelFunc func()) {
	v.bus.Publish(event.New(editor.NewCloseRequested(e, cancelFunc)))
	fyne.Do(e.Window().RequestFocus)
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
		v.bus.Publish(evt.NewFollowup(editor.LoadFailed{
			Editor: e,
			Err:    err,
		}))
		return
	}

	v.mu.Lock()
	v.loadedContents[pl.File.FullPath()] = pl.Content
	v.mu.Unlock()

	v.bus.Publish(evt.NewFollowup(editor.Loaded{
		Editor:  e,
		Content: pl.Content,
	}))
}

func (v *editorViewModelImpl) handleFileLoadingFailure(evt event.Event) {
	pl := evt.Payload().(directory.LoadFileFailed)
	v.notifier.NotifyError(pl.Err)

	e, ok := v.openedEditors[pl.File.FullPath()]
	if !ok {
		// The editor has been closed before the file was loaded. And it's okay
		return
	}

	v.bus.Publish(evt.NewFollowup(editor.LoadFailed{
		Editor: e,
		Err:    pl.Err,
	}))
}

func (v *editorViewModelImpl) handleEditorCloseConfirmed(evt event.Event) {
	pl := evt.Payload().(editor.CloseConfirmed)
	fyne.Do(pl.Editor.Window().Close)
	v.unregisterEditor(pl.Editor.File())
	v.bus.Publish(evt.NewFollowup(editor.Closed(pl)))
}

func (v *editorViewModelImpl) handleEditorCloseCanceled(evt event.Event) {
	pl := evt.Payload().(editor.CloseCanceled)
	fyne.Do(pl.Editor.Window().RequestFocus)
}

func (v *editorViewModelImpl) IsOpen(file *directory.File) bool {
	_, ok := v.openedEditors[file.FullPath()]
	return ok
}

func (v *editorViewModelImpl) unregisterEditor(file *directory.File) {
	path := file.FullPath()
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.openedEditors, path)
	delete(v.loadedContents, path)
}

func (v *editorViewModelImpl) closeAll() {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, oe := range v.openedEditors {
		v.requestClose(oe, func() {}) // TODO: move this when the connection change is triggered and warn the user for unsaved changes before closing the editors
	}
}

func (v *editorViewModelImpl) forceCloseAll() {
	for _, oe := range v.openedEditors {
		fyne.Do(oe.Window().Close)
	}
}
