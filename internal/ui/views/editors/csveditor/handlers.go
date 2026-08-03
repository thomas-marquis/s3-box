package csveditor

import (
	"encoding/csv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

func (e *csvEditor) handleLoaded(evt event.Event) {
	pl := evt.Payload().(editor.Loaded)

	e.IsLoading.Set(false) //nolint:errcheck

	r := csv.NewReader(pl.Content)

	nbRows := 0
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		e.Records.Append(record) //nolint:errcheck
		nbRows++
	}

	if e.Records.Length() == 0 {
		return
	}

	e.updateContentHash(e.getContent())

	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)

	firstRow, _ := e.Records.GetValue(0)
	nbCols := len(firstRow)
	for i := range nbCols {
		col := csvColumn{}
		for j := range nbRows {
			row, _ := e.Records.GetValue(j)
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
		e.Bus.Publish(pl.Cancel(evt))
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
