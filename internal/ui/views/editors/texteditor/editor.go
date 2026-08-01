package texteditor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type textEditor struct {
	editor.Base

	Content binding.String

	ConfirmClose func(onConfirm func(confirmed bool))

	mu                   sync.Mutex
	contentHash          string
	cancelFunc           func()
	shouldCloseWhenSaved bool

	sub *event.Subscriber
}

func New(bus event.Bus, window fyne.Window, file *directory.File) editor.Editor {
	e := &textEditor{
		Base:    editor.NewBase(bus, window, file),
		Content: binding.NewString(),
	}

	e.ExtendBaseEditor(e)

	e.IsLoading.Set(true) //nolint:errcheck

	e.Sub.
		On(event.Is(editor.LoadedType), e.handleLoaded).
		On(event.Is(editor.LoadFailedType), e.handleLoadFailed).
		On(event.Is(editor.CloseRequestedType), e.handleCloseRequested)
	e.Sub.ListenWithWorkers(2)

	return e
}

func (e *textEditor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *textEditor) RequestClose() {
	e.Bus.Publish(event.New(editor.CloseRequested{
		Editor: e,
		Cancel: func() {},
	}))
}

func (e *textEditor) Save(content string) {
	e.IsLoading.Set(true)          //nolint:errcheck
	e.StatusLabel.Set("Saving...") // nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.cancelFunc = cancel
	e.mu.Unlock()

	e.Bus.Publish(event.New(editor.SaveTriggered{
		File:    e.File(),
		Content: content,
	}, event.WithContext(ctx)))
}

func (e *textEditor) SaveThenExit(content string) {
	e.shouldCloseWhenSaved = true
	e.Save(content)
}

func (e *textEditor) OnSaved(newContent string, err error) {
	e.IsLoading.Set(false) //nolint:errcheck
	defer e.Cancel()

	if err != nil {
		e.StatusLabel.Set("error (unsaved)") //nolint:errcheck
		e.Err.Set(err)                       //nolint:errcheck

		e.mu.Lock()
		e.shouldCloseWhenSaved = false
		e.mu.Unlock()
		return
	}

	e.updateContentHash(newContent)
	e.StatusLabel.Set(fmt.Sprintf("Saved %s", time.Now().Format("15:04:05"))) // nolint:errcheck
	e.mu.Lock()
	if e.shouldCloseWhenSaved {
		e.Bus.Publish(event.New(editor.CloseConfirmed{
			Editor: e,
		}))
	}
	e.mu.Unlock()
}

//func (e *textEditor) Close() bool {
//	if e.HasChanged() {
//		return false
//	}
//	e.Cancel()
//	e.Window().Close() // force close
//	return true
//}

func (e *textEditor) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancelFunc == nil {
		return
	}
	e.cancelFunc()
	e.cancelFunc = nil
}

func (e *textEditor) HasChanged() bool {
	val, _ := e.Content.Get()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.contentHash != sha256Hex(val)
}

func (e *textEditor) updateContentHash(newContent string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.contentHash = sha256Hex(newContent)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
