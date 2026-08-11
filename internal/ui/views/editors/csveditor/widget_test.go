package csveditor_test

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	csvContent = `id,name,age
1,toto,12
2,lolo,13`
	csvLargeContent = `id,name,age
1,toto,12
2,lolo,13
3,alice,25
4,bob,31
5,charlie,28
6,diana,22
7,edward,45
8,fiona,33
9,george,29
10,hannah,27
11,isaac,38
12,julia,24
13,kevin,41
14,laura,26
15,michael,35
16,nancy,30
17,oliver,23`
)

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
	f.file, _ = directory.NewFile("test.csv", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	f.window = fyne_test.NewWindow(nil)
	f.window.Resize(fyne.NewSize(500, 300))
	f.editor = csveditor.New(f.bus, f.window, f.file)

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

func TestCsvEditorWidget(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	t.Run("should display the csv content when loaded successfully", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		widgt := ed.CreateWidget().(*csveditor.Widget)
		ed.Window().SetContent(widgt)
		canvas := ed.Window().Canvas()

		mockContent := &directory.InMemoryContent{
			Data: []byte(csvContent),
		}

		testutil.AssertImageMatches(t, "images/is-loading.png", canvas.Capture())

		// When
		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: mockContent,
		}))

		time.Sleep(300 * time.Millisecond) // Magic wait...
		widgt.Refresh()

		// Then
		testutil.AssertImageMatches(t, "images/loaded-successfully.png", canvas.Capture())
	})

	t.Run("should display page 2", func(t *testing.T) {
		// Given
		fxt := setup(t)
		ed := fxt.Editor()

		widgt := ed.CreateWidget().(*csveditor.Widget)
		ed.Window().SetContent(widgt)
		canvas := ed.Window().Canvas()

		mockContent := &directory.InMemoryContent{
			Data: []byte(csvLargeContent),
		}

		fxt.Bus().Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: mockContent,
		}))

		time.Sleep(300 * time.Millisecond) // Magic wait...
		widgt.Refresh()

		testutil.AssertImageMatches(t, "images/page-1.png", canvas.Capture())

		// When - go to page 2
		fyne_test.Tap(widgt.NextBtn)

		time.Sleep(300 * time.Millisecond) // Magic wait...
		widgt.Refresh()

		// Then
		testutil.AssertImageMatches(t, "images/page-2.png", canvas.Capture())

		// When - go back page 1
		fyne_test.Tap(widgt.PrevBtn)

		time.Sleep(300 * time.Millisecond) // Magic wait...
		widgt.Refresh()

		// Then
		testutil.AssertImageMatches(t, "images/page-1.png", canvas.Capture())
	})
}
