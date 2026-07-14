package texteditor_test

import (
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	fyne_test "fyne.io/fyne/v2/test"
	fyne_widget "fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	"go.uber.org/mock/gomock"
)

// uv run ./tools/diff_images.py --folders internal/ui/views/editors/texteditor/testdata/images internal/ui/views/editors/texteditor/testdata/failed/images --color "red"

var (
	lastModified = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
)

func TestTextEditor_saving(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	fyne_test.NewApp()

	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.txt", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should save updated content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockBus.EXPECT().Publish(gomock.Any()).AnyTimes()

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))
		ed := texteditor.New(mockBus, w, file)

		res := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := w.Canvas()
		canvas.SetContent(res)

		// When & Then
		fyne_test.Type(res.TextEntry, "my new content")
		fyne_test.Tap(res.SaveBtn.ToolbarObject().(*fyne_widget.Button))
		testutil.AssertImageMatches(t, "images/updated-and-saving.png", canvas.Capture())

		ed.OnSaved("my new content", nil)
		testutil.AssertImageMatches(t, "images/updated-and-saved.png", canvas.Capture())
	})

	t.Run("should display save error", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockBus.EXPECT().Publish(gomock.Any()).AnyTimes()

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))
		ed := texteditor.New(mockBus, w, file)

		wdg := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := w.Canvas()
		canvas.SetContent(wdg)

		// When
		fyne_test.Type(wdg.TextEntry, "my new content")
		fyne_test.Tap(wdg.SaveBtn.ToolbarObject().(*fyne_widget.Button))

		ed.OnSaved("", errors.New("failed to save"))

		// Then
		testutil.AssertImageMatches(t, "images/saving-error.png", canvas.Capture())
	})
}

func TestTextEditor_loading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	fyne_test.NewApp()

	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.txt", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should display empty content when file is not loaded yet", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))

		ed := texteditor.New(mockBus, w, file)

		// When
		res := ed.CreateWidget()
		w.Canvas().SetContent(res)

		// Then
		testutil.AssertImageMatches(t, "images/is-loading.png", w.Canvas().Capture())
	})

	t.Run("should display file content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))

		ed := texteditor.New(mockBus, w, file)
		mockContent := &directory.InMemoryContent{
			Data: []byte("Hello world!"),
			Pos:  0,
		}

		// When
		res := ed.CreateWidget()
		w.Canvas().SetContent(res)

		ed.OnLoaded(mockContent, nil)

		// Then
		testutil.AssertImageMatches(t, "images/loaded-with-content.png", w.Canvas().Capture())
	})

	t.Run("should display error message", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))

		ed := texteditor.New(mockBus, w, file)

		// When
		res := ed.CreateWidget()
		w.Canvas().SetContent(res)

		ed.OnLoaded(nil, errors.New("error loading file"))

		// Then
		testutil.AssertImageMatches(t, "images/loaded-with-error.png", w.Canvas().Capture())
	})
}

func TestTextEditor_close(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	fyne_test.NewApp()

	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.txt", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should warn user when file has been updated", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))

		ed := texteditor.New(mockBus, w, file)
		wdg := ed.CreateWidget().(*texteditor.TextEditor)
		canvas := w.Canvas()
		canvas.SetContent(wdg)

		mockContent := &directory.InMemoryContent{
			Data: []byte("Hello "),
			Pos:  0,
		}
		ed.OnLoaded(mockContent, nil)

		// When
		fyne_test.Type(wdg.TextEntry, "World!")
		wdg.TextEntry.TypedShortcut(&desktop.CustomShortcut{
			KeyName:  fyne.KeyQ,
			Modifier: fyne.KeyModifierControl,
		})

		// Then
		testutil.AssertImageMatches(t, "images/close-with-unsave.png", w.Canvas().Capture())
	})
}
