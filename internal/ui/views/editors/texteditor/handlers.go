package texteditor

import (
	"io"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

func (e *textEditor) handleLoaded(evt event.Event) {
	pl := evt.Payload().(editor.Loaded)

	e.IsLoading.Set(false) //nolint:errcheck

	contentVal, err := io.ReadAll(pl.Content)
	if err != nil {
		e.Err.Set(err) //nolint:errcheck
		return
	}

	strContent := string(contentVal)
	e.updateContentHash(strContent)

	e.Content.Set(strContent) //nolint:errcheck
}

func (e *textEditor) handleLoadFailed(evt event.Event) {
	pl := evt.Payload().(editor.LoadFailed)
	e.StatusLabel.Set("error (unloaded)") //nolint:errcheck
	e.IsLoading.Set(false)                //nolint:errcheck
	e.Err.Set(pl.Err)                     //nolint:errcheck
}

func (e *textEditor) handleCloseRequested(evt event.Event) {
	pl := evt.Payload().(editor.CloseRequested)

	if !e.HasChanged() {
		pl.Cancel()
		e.Bus.Publish(evt.NewFollowup(editor.CloseConfirmed{Editor: e}))
		return
	}

	e.ConfirmClose(func(confirmed bool) {
		if confirmed {
			pl.Cancel()
			e.Bus.Publish(evt.NewFollowup(editor.CloseConfirmed{Editor: e}))
		} else {
			e.Bus.Publish(evt.NewFollowup(editor.CloseCanceled{Editor: e}))
		}
	})
}
