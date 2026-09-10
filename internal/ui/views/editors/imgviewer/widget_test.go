package imgviewer_test

import (
	"context"
	"embed"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/imgviewer"
)

//go:embed testdata/images/*
var testImages embed.FS

// uv run ./tools/diff_images.py --folders internal/ui/views/editors/imgviewer/testdata/images internal/ui/views/editors/imgviewer/testdata/failed/images --color "red"

var (
	lastModified = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
)

type fixture struct {
	bus    event.Bus
	ctx    context.Context
	cancel context.CancelFunc
	t      *testing.T
	editor editor.Editor
	file   *directory.File
	window fyne.Window
	app    fyne.App
}

func setup(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t}
	f.app = fyne_test.NewApp()

	f.ctx, f.cancel = context.WithCancel(context.Background())
	f.bus = inmemory.NewBus(f.ctx)

	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	f.file, _ = directory.NewFile("test.png", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	f.window = fyne_test.NewWindow(nil)
	f.window.Resize(fyne.NewSize(500, 300))
	f.editor = imgviewer.New(f.bus, f.window, f.file)

	t.Cleanup(f.teardown)

	return f
}

func (f *fixture) App() fyne.App {
	f.t.Helper()
	return f.app
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

func TestImageViewerWidget(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	t.Run("should display loading state initially", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		widget := ed.CreateWidget()
		ed.Window().SetContent(widget)
		canvas := ed.Window().Canvas()

		// When & Then - should initially show loading state
		tu.AssertImageMatches(t, "images/is-loading.png", canvas.Capture())
	})

	t.Run("should display image when loaded", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		widget := ed.CreateWidget()
		ed.Window().SetContent(widget)
		canvas := ed.Window().Canvas()

		// Verify initial loading state
		tu.AssertImageMatches(t, "images/is-loading.png", canvas.Capture())

		pngImage := getTestPNGImage()
		mockContent := &directory.InMemoryContent{
			Data: pngImage,
		}

		// When
		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: mockContent,
		}))

		// Wait for the widget to process the event
		time.Sleep(100 * time.Millisecond)

		// Then
		tu.AssertImageMatches(t, "images/loaded.png", canvas.Capture())
	})

	t.Run("should display error state when loading fails", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		widget := ed.CreateWidget()
		ed.Window().SetContent(widget)
		canvas := ed.Window().Canvas()

		// Verify initial loading state
		tu.AssertImageMatches(t, "images/is-loading.png", canvas.Capture())

		// When
		fxt.Bus().Publish(event.New(editor.LoadFailed{
			Editor: ed,
			Err:    errors.New("failed to load image"),
		}))

		// Wait for the widget to process the event
		time.Sleep(100 * time.Millisecond)

		// Then
		tu.AssertImageMatches(t, "images/load-error.png", canvas.Capture())
	})
}

// getTestPNGImage loads and returns the test PNG image bytes
func getTestPNGImage() []byte {
	data, err := testImages.ReadFile("testdata/images/test-image.png")
	if err != nil {
		panic("failed to load test image: " + err.Error())
	}
	return data
}
