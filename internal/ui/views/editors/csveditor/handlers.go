package csveditor

import (
	"encoding/csv"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

func (e *Editor) handleLoaded(evt event.Event) {
	defer u.SkipD1(e.IsLoading.Set, false)

	pl := evt.Payload().(editor.Loaded)

	r := csv.NewReader(pl.Content)

	nbRows := 0
	e.Paginator.Records = nil
	e.Paginator.CurrentIndex = 0
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		nbRows++
		e.Paginator.Append(record)
	}

	if len(e.Paginator.Records) == 0 {
		return
	}

	e.updateContentHash(e.GetContent())
	e.SetContent(pl.Content)
	e.UpdatePageLabel()
	e.updateColumnsWidth()
}

func (e *Editor) handleLoadFailed(evt event.Event) {
	pl := evt.Payload().(editor.LoadFailed)
	u.Skip(e.IsLoading.Set(false))
	u.Skip(e.StatusLabel.Set("error (unloaded)"))
	u.Skip(e.Err.Set(pl.Err))
}

func (e *Editor) handleCloseRequested(evt event.Event) {
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
