package csveditor

import (
	"encoding/csv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

func (e *csvEditor) handleLoaded(evt event.Event) {
	defer e.IsLoading.Set(false) //nolint:errcheck

	pl := evt.Payload().(editor.Loaded)

	r := csv.NewReader(pl.Content)

	nbRows := 0
	paginator := NewCsvPaginator(e.Records)
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		nbRows++
		paginator.Append(record)
	}

	if len(paginator.Records) == 0 {
		return
	}

	e.updateContentHash(e.getContent())
	e.SetContent(pl.Content)

	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)

	firstRow := paginator.Records[0]
	nbCols := len(firstRow)
	for i := range nbCols {
		col := csvColumn{}
		for j := range nbRows {
			row := paginator.Records[j]
			cw := colWidth(row[i], textSize)
			if col.Width < cw-cellPadding {
				col.Width = cw
			}
		}
		e.Columns.Append(col) //nolint:errcheck
	}
}

func (e *csvEditor) handleLoadFailed(evt event.Event) {
	pl := evt.Payload().(editor.LoadFailed)
	e.IsLoading.Set(false)                //nolint:errcheck
	e.StatusLabel.Set("error (unloaded)") //nolint:errcheck
	e.Err.Set(pl.Err)                     //nolint:errcheck
}

func (e *csvEditor) handleCloseRequested(evt event.Event) {
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
