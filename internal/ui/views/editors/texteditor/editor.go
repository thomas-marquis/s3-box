package texteditor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type textEditor struct {
	*editor.Base

	ContentStr binding.String

	ConfirmClose func(onConfirm func(confirmed bool))

	contentHash          string
	cancelFunc           func()
	shouldCloseWhenSaved bool
}

func New(bus event.Bus, window fyne.Window, file *directory.File) editor.Editor {
	e := &textEditor{
		Base:       editor.NewBase(bus, window, file),
		ContentStr: binding.NewString(),
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

func (e *textEditor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *textEditor) Save(content string) {
	u.Skip(e.IsLoading.Set(true))
	u.Skip(e.StatusLabel.Set("Saving..."))

	ctx, cancel := context.WithCancel(context.Background())

	e.Lock()
	e.cancelFunc = cancel
	e.Unlock()

	handleFailure := func(err error) {
		u.Skip(e.StatusLabel.Set("error (unsaved)"))
		u.Skip(e.Err.Set(err))

		e.Lock()
		e.shouldCloseWhenSaved = false
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
		e.Lock()
		if e.shouldCloseWhenSaved {
			e.RequestClose()
		}
		e.Unlock()
	}()
}

func (e *textEditor) SaveThenExit(content string) {
	e.Lock()
	defer e.Unlock()
	e.shouldCloseWhenSaved = true
	e.Save(content)
}

func (e *textEditor) RequestClose() {
	e.Bus.Publish(event.New(editor.CloseRequested{
		Editor: e,
	}))
}

func (e *textEditor) Cancel() {
	e.Lock()
	defer e.Unlock()

	if e.cancelFunc == nil {
		return
	}
	e.cancelFunc()
	e.cancelFunc = nil
}

func (e *textEditor) HasChanged() bool {
	val, _ := e.ContentStr.Get()
	e.Lock()
	defer e.Unlock()
	return e.contentHash != sha256Hex(val)
}

func (e *textEditor) updateContentHash(newContent string) {
	e.Lock()
	defer e.Unlock()
	e.contentHash = sha256Hex(newContent)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s)) // [32]byte
	return hex.EncodeToString(sum[:])
}
