package imgviewer

import (
	"bytes"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type Editor struct {
	*editor.Base

	cancelFunc func()
	ImageData  binding.Item[[]byte]
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	e := &Editor{
		Base:      editor.NewBase(bus, w, file),
		ImageData: binding.NewItem[[]byte](bytes.Equal),
	}

	e.ExtendBaseEditor(e)

	u.Skip(e.IsLoading.Set(true))

	e.Sub.
		On(event.Is(editor.LoadedType), e.handleLoaded).
		On(event.Is(editor.LoadFailedType), e.handleLoadFailed).
		On(event.Is(editor.CloseRequestedType), e.handleCloseRequested)
	e.Sub.ListenWithWorkers(2)

	return e
}

func (e *Editor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *Editor) RequestClose() {
	e.Bus.Publish(event.New(editor.CloseRequested{
		Editor: e,
	}))
}

func (e *Editor) Cancel() {
	e.Lock()
	defer e.Unlock()

	if e.cancelFunc == nil {
		return
	}
	e.cancelFunc()
	e.cancelFunc = nil
}

func (e *Editor) HasChanged() bool {
	return false // Image viewer is read-only
}

func (e *Editor) handleLoaded(evt event.Event) {
	defer u.SkipD1(e.IsLoading.Set, false)
	pl := evt.Payload().(editor.Loaded)

	contentVal, err := io.ReadAll(pl.Content)
	if err != nil {
		u.Skip(e.Err.Set(err))
		return
	}

	e.SetContent(pl.Content)
	u.Skip(e.ImageData.Set(contentVal))
	u.Skip(e.StatusLabel.Set("Loaded"))
}

func (e *Editor) handleLoadFailed(evt event.Event) {
	pl := evt.Payload().(editor.LoadFailed)
	u.Skip(e.StatusLabel.Set("error (unloaded)"))
	u.Skip(e.IsLoading.Set(false))
	u.Skip(e.Err.Set(pl.Err))
}

func (e *Editor) handleCloseRequested(evt event.Event) {
	pl := evt.Payload().(editor.CloseRequested)
	e.Bus.Publish(pl.Confirm(evt))
}
