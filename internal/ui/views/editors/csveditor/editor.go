package csveditor

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
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

	TableBinding *uu.TableBinding[string]
	Columns      binding.List[ColWidth]
	Paginator    *Paginator
	IsLazy       bool
	PageLabel    binding.String
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	return newEditor(bus, w, file)
}

func NewLazy(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	e := newEditor(bus, w, file)
	e.IsLazy = true
	return e
}

func newEditor(bus event.Bus, w fyne.Window, file *directory.File) *Editor {
	ed := &Editor{
		PageLabel: binding.NewString(),
		Base:      editor.NewBase(bus, w, file),
		TableBinding: uu.NewTableBinding[string](func(a, b string) bool {
			return a == b
		}),
		Columns: binding.NewList(func(c1, c2 ColWidth) bool {
			return cmp.Compare(c1, c2) == 0
		}),
		dataListeners: make(map[string]func()),
	}
	ed.Paginator = NewCsvPaginator(ed.TableBinding)
	ed.Paginator.HasHeader.AddListener(binding.NewDataListener(ed.updateColumnsWidth))

	ed.ExtendBaseEditor(ed)

	u.Skip(ed.IsLoading.Set(true))

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

	u.Skip(e.IsLoading.Set(true))
	defer u.SkipD1(e.IsLoading.Set, false)

	hasMore := e.Paginator.Next()
	e.UpdatePageLabel()
	e.updateColumnsWidth()
	return hasMore
}

func (e *Editor) PrevPage() {
	if e.Paginator.CurrentIndex() == 0 {
		return
	}

	u.Skip(e.IsLoading.Set(true))
	defer u.SkipD1(e.IsLoading.Set, false)

	e.Paginator.Prev()
	e.UpdatePageLabel()
	e.updateColumnsWidth()
}

func (e *Editor) UpdatePageLabel() {
	totalPages := e.Paginator.TotalPages()
	if totalPages < 0 {
		// Lazy loading: total pages unknown
		u.Skip(
			e.PageLabel.Set(
				fmt.Sprintf("%d / ?", e.Paginator.PageNumber())),
		)
	} else {
		u.Skip(
			e.PageLabel.Set(
				fmt.Sprintf("%d / %d", e.Paginator.PageNumber(), totalPages)),
		)
	}
}

func (e *Editor) Save() {
	u.Skip(e.IsLoading.Set(true))
	u.Skip(e.StatusLabel.Set("Saving..."))

	content := e.GetContent()
	ctx, cancel := context.WithCancel(context.Background())

	e.Lock()
	e.cancelFunc = cancel
	e.Unlock()

	handleFailure := func(err error) {
		u.Skip(e.StatusLabel.Set("error (unsaved)"))
		u.Skip(e.Err.Set(err))

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
		defer u.SkipD1(e.IsLoading.Set, false)

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
		u.Skip(
			e.StatusLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05"))),
		)
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
	rows, cols := e.TableBinding.Dims()
	if rows == 0 || cols == 0 {
		return ""
	}

	builder := strings.Builder{}
	for i := range rows{
		for j := range cols {
			val, err := e.TableBinding.ValueAt(i, j)
			if err != nil {
				return ""
			}
			builder.WriteString(val)
			if j < cols-1 {
				builder.WriteString(sep)
			}
		}
		if i < rows-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

const fileSizeROThreshold = 5 * 1024 // TODO: define it in factory

func (e *Editor) IsReadOnly() bool {
	return e.File().SizeBytes() > fileSizeROThreshold
}

func (e *Editor) updateContentHash(newContent string) {
	e.Lock()
	defer e.Unlock()
	e.contentHash = sha256Hex(newContent)
}

func (e *Editor) updateColumnsWidth() {
	rows, cols := e.TableBinding.Dims()
	if rows == 0 || cols == 0 {
		return
	}
	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)

	var colWidths []ColWidth
	hasHeader := u.SkipV(e.Paginator.HasHeader.Get())

	// Calculate column widths based on all visible rows
	for i := 0; i < cols; i++ {
		col := colMinWidth
		
		// Check all rows in the current table (which has been resized to current page)
		for j := 0; j < rows; j++ {
			// Check if this is the header row
			isHeaderRow := hasHeader && j == 0
			
			val, err := e.TableBinding.ValueAt(j, i)
			if err != nil {
				continue
			}
			cw := colWidth(val, textSize)
			
			// For header row, use it as-is
			if isHeaderRow {
				col = ColWidth(cw)
			} else {
				// For data rows, compare with current max
				if float32(col) < cw-cellPadding {
				col = ColWidth(cw)
				}
			}
			
			if col >= colMaxWidth {
				col = colMaxWidth
				break
			}
		}
		colWidths = append(colWidths, col)
	}

	u.Skip(e.Columns.Set(colWidths))
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
