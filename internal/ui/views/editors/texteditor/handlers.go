package texteditor

import (
	"io"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

func (e *textEditor) handleLoaded(evt event.Event) {
	defer u.SkipD1(e.IsLoading.Set, false)
	pl := evt.Payload().(editor.Loaded)

	contentVal, err := io.ReadAll(pl.Content)
	if err != nil {
		u.Skip(e.Err.Set(err))
		return
	}

	strContent := string(contentVal)
	e.updateContentHash(strContent)
	e.SetContent(pl.Content)

	u.Skip(e.ContentStr.Set(strContent))
}

func (e *textEditor) handleLoadFailed(evt event.Event) {
	pl := evt.Payload().(editor.LoadFailed)
	u.Skip(e.StatusLabel.Set("error (unloaded)"))
	u.Skip(e.IsLoading.Set(false))
	u.Skip(e.Err.Set(pl.Err))
}

func (e *textEditor) handleCloseRequested(evt event.Event) {
	pl := evt.Payload().(editor.CloseRequested)

	if !e.HasChanged() {
		e.Bus.Publish(pl.Confirm(evt))
		return
	}

	e.ConfirmClose(func(confirmed bool) {
		if confirmed {
			e.Bus.Publish(pl.Confirm(evt))
		} else {
			e.Bus.Publish(pl.Cancel(evt))
		}
	})
}
