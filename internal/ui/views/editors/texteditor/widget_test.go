package texteditor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	fyne_widget "fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"
)

// uv run ./tools/diff_images.py --folders internal/ui/views/editors/texteditor/testdata/images internal/ui/views/editors/texteditor/testdata/failed/images --color "red"

var (
	lastModified = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
)

type fakeFileContent struct {
	*directory.InMemoryContent
	Written chan struct{}
	Readed  chan struct{}
}

var (
	_ directory.FileContent = (*fakeFileContent)(nil)
)

func (c *fakeFileContent) Read(buff []byte) (int, error) {
	<-c.Readed
	return c.InMemoryContent.Read(buff)
}

func (c *fakeFileContent) Write(content []byte) (int, error) {
	<-c.Written
	return c.InMemoryContent.Write(content)
}

type fakeFileContentWithWriteErr struct {
	*directory.InMemoryContent
	err error
}

func (c *fakeFileContentWithWriteErr) Write(content []byte) (int, error) {
	return 0, c.err
}

type fixture struct {
	bus    event.Bus
	ctx    context.Context
	cancel context.CancelFunc
	t      *testing.T
	editor editor.Editor
	file   *directory.File
	window fyne.Window
}

func setup(t *testing.T) *fixture {
	t.Helper()
	fyne_test.NewApp()
	f := &fixture{t: t}

	f.ctx, f.cancel = context.WithCancel(context.Background())
	f.bus = inmemory.NewBus(f.ctx)

	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	f.file, _ = directory.NewFile("test.txt", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	f.window = fyne_test.NewWindow(nil)
	f.window.Resize(fyne.NewSize(500, 300))
	f.editor = texteditor.New(f.bus, f.window, f.file)

	t.Cleanup(f.teardown)

	return f
}

func (f *fixture) Window() fyne.Window {
	f.t.Helper()
	return f.window
}

func (f *fixture) File() *directory.File {
	f.t.Helper()
	return f.file
}

func (f *fixture) Editor() editor.Editor {
	f.t.Helper()
	return f.editor
}

func (f *fixture) Bus() event.Bus {
	f.t.Helper()
	return f.bus
}

func (f *fixture) teardown() {
	f.cancel()
}

func TestTextEditor_CreateWidget(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	t.Run("should save updated content", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		res := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := fxt.Window().Canvas()
		canvas.SetContent(res)

		written := make(chan struct{})
		readed := make(chan struct{})

		content := &fakeFileContent{
			InMemoryContent: &directory.InMemoryContent{
				Data: make([]byte, 0),
			},
			Written: written,
			Readed:  readed,
		}

		// When the file is loading
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/is-loading.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)

		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		close(readed) // Simulate end loading
		// When the file is loaded
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/loaded-empty.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)

		// When user types some text, then save
		fyne_test.Type(res.TextEntry, "my new content")
		fyne_test.Tap(res.SaveBtn.ToolbarObject().(*fyne_widget.Button))

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/updated-and-saving.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)

		close(written)
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/updated-and-saved.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should load and display non empty file", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		res := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := fxt.Window().Canvas()
		canvas.SetContent(res)

		content := &directory.InMemoryContent{
			Data: []byte("something crazy"),
		}

		// When
		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Then
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/loaded-non-empty.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should display save error when loading failed", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		res := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := fxt.Window().Canvas()
		canvas.SetContent(res)

		// When
		fxt.Bus().Publish(event.New(editor.LoadFailed{
			Editor: ed,
			Err:    errors.New("summer is coming"),
		}))

		// Then
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/loaded-error.png", fxt.Window().Canvas().Capture())
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should display the error message when saving fails", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		res := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := fxt.Window().Canvas()
		canvas.SetContent(res)

		content := &fakeFileContentWithWriteErr{
			InMemoryContent: &directory.InMemoryContent{
				Data: make([]byte, 0),
			},
			err: errors.New("the world just ended"),
		}

		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Wait until the file is loaded
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/loaded-empty.png", fxt.Window().Canvas().Capture())
		}, time.Second, 10*time.Millisecond)

		// When user types some text, then save
		fyne_test.Type(res.TextEntry, "my new content")
		fyne_test.Tap(res.SaveBtn.ToolbarObject().(*fyne_widget.Button))

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/updated-and-saved-with-error.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should display a message when closed without saving", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		res := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := fxt.Window().Canvas()
		canvas.SetContent(res)

		content := &directory.InMemoryContent{
			Data: make([]byte, 0),
		}

		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Wait until the file is loaded
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/loaded-empty.png", fxt.Window().Canvas().Capture())
		}, time.Second, 10*time.Millisecond)

		fyne_test.Type(res.TextEntry, "my new content")

		// When
		fxt.Bus().Publish(event.New(editor.CloseRequested{Editor: ed}))

		// Then
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			testutil.AssertImageMatches(ct, "images/updated-not-saved-and-closed.png", canvas.Capture())
		}, time.Second, 10*time.Millisecond)
	})
}
