package csveditor

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	sep                     = ","
	listenerColumnsWidthKey = "listener.columns.width"
)

var (
	shortcutSave = desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierControl,
	}
)

type ColWidth float32

type Editor struct {
	*editor.Base

	cancelFunc  func()
	contentHash string
	// dataListeners is necessary to notify the UI that the editor's state has changed.
	// In some cases, we can't rely on the binding's listeners.
	// For example, for binding.List, the event listeners are triggered only if the list size has changed...
	dataListeners map[string]func()

	Records   binding.List[[]string]
	Columns   binding.List[ColWidth]
	Paginator *Paginator

	PageLabel binding.String
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	ed := &Editor{
		PageLabel: binding.NewString(),
		Base:      editor.NewBase(bus, w, file),
		Records:   binding.NewList[[]string](slices.Equal),
		Columns: binding.NewList[ColWidth](func(c1, c2 ColWidth) bool {
			return cmp.Compare(c1, c2) == 0
		}),
		dataListeners: make(map[string]func()),
	}
	ed.Paginator = NewCsvPaginator(ed.Records)

	ed.ExtendBaseEditor(ed)

	ed.IsLoading.Set(true) //nolint:errcheck

	w.Canvas().AddShortcut(&shortcutSave, func(fyne.Shortcut) {
		ed.Save()
	})

	ed.Sub.
		On(event.Is(editor.LoadedType), ed.handleLoaded).
		On(event.Is(editor.LoadFailedType), ed.handleLoadFailed).
		On(event.Is(editor.CloseRequestedType), ed.handleCloseRequested)
	ed.Sub.ListenWithWorkers(2)

	return ed
}

func (e *Editor) AddListener(name string, listener func()) {
	e.dataListeners[name] = listener
}

func (e *Editor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *Editor) NextPage() bool {
	if !e.Paginator.HasNext() {
		return false
	}

	e.IsLoading.Set(true)        //nolint:errcheck
	defer e.IsLoading.Set(false) //nolint:errcheck

	hasMore := e.Paginator.Next()
	e.UpdatePageLabel()
	e.updateColumnsWidth()
	return hasMore
}

func (e *Editor) PrevPage() {
	if e.Paginator.CurrentIndex == 0 {
		return
	}

	e.IsLoading.Set(true)        //nolint:errcheck
	defer e.IsLoading.Set(false) //nolint:errcheck

	e.Paginator.Prev()
	e.UpdatePageLabel()
	e.updateColumnsWidth()
}

func (e *Editor) UpdatePageLabel() {
	e.PageLabel.Set(fmt.Sprintf("%d / %d", e.Paginator.PageNumber(), e.Paginator.TotalPages())) //nolint:errcheck
}

func (e *Editor) Save() {
	e.IsLoading.Set(true)          //nolint:errcheck
	e.StatusLabel.Set("Saving...") // nolint:errcheck

	content := e.GetContent()
	ctx, cancel := context.WithCancel(context.Background())

	e.Lock()
	e.cancelFunc = cancel
	e.Unlock()

	handleFailure := func(err error) {
		e.StatusLabel.Set("error (unsaved)") //nolint:errcheck
		e.Err.Set(err)                       //nolint:errcheck

		e.Lock()
		e.cancelFunc = nil
		e.Unlock()
	}

	if !e.IsLoaded() {
		handleFailure(errors.New("file not loaded"))
		return
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			e.Content.Cancel()
			close(done)
		case <-done:
		}
	}()

	go func() {
		defer close(done)
		defer e.IsLoading.Set(false) //nolint:errcheck

		if _, err := e.Content.Seek(0, io.SeekStart); err != nil {
			handleFailure(err)
			return
		}
		if _, err := fmt.Fprint(e.Content, content); err != nil {
			handleFailure(err)
			return
		}
		e.File().SetSizeBytes(uint64(len(content)))

		e.updateContentHash(content)
		e.StatusLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05"))) // nolint:errcheck
	}()
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
	e.Lock()
	defer e.Unlock()
	return e.contentHash != sha256Hex(e.GetContent())
}

func (e *Editor) GetContent() string {
	if len(e.Paginator.Records) == 0 {
		return ""
	}

	builder := strings.Builder{}
	for i, row := range e.Paginator.Records {
		for j, cell := range row {
			builder.WriteString(cell)
			if j < len(row)-1 {
				builder.WriteString(sep)
			}
		}
		if i < len(e.Paginator.Records)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func (e *Editor) updateContentHash(newContent string) {
	e.Lock()
	defer e.Unlock()
	e.contentHash = sha256Hex(newContent)
}

func (e *Editor) updateColumnsWidth() {
	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)

	firstVisibleRow := e.Paginator.Records[e.Paginator.CurrentIndex]
	nbCols := len(firstVisibleRow)
	var colWidths []ColWidth
	for i := range nbCols {
		col := colMinWidth
		for j := range e.Paginator.CurrentPageSize() {
			row := e.Paginator.Records[e.Paginator.CurrentIndex+j]
			cw := colWidth(row[i], textSize)
			if float32(col) < cw-cellPadding {
				col = ColWidth(cw)
			}
			if col >= colMaxWidth {
				col = colMaxWidth
				break
			}
		}
		colWidths = append(colWidths, col)
	}

	e.Columns.Set(colWidths) //nolint:errcheck
	if listener, ok := e.dataListeners[listenerColumnsWidthKey]; ok {
		listener()
	}
}

func colWidth(text string, textSize float32) float32 {
	return fyne.MeasureText(text, textSize, fyne.TextStyle{}).Width + cellPadding
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
