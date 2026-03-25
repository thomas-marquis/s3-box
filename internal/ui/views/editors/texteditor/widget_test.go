package texteditor_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/fileeditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	"go.uber.org/mock/gomock"
)

// uv run ./tools/diff_images.py --folders internal/ui/views/editors/texteditor/testdata/images internal/ui/views/editors/texteditor/testdata/failed/images --color "red"

func TestFileEditor_loading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	fyne_test.NewApp()

	lastModified := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.txt", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should display empty content when file is not loaded yet", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))
		es := &fileeditor.State{
			Window:   w,
			File:     file,
			Content:  binding.NewString(),
			IsLoaded: binding.NewBool(),
			ErrorMsg: binding.NewString(),
			Bus:      mockBus,
		}

		// When
		res := texteditor.NewTextEditor(es)
		w.Canvas().SetContent(res)

		// Then
		testutil.AssertImageMatches(t, "images/is-loading.png", w.Canvas().Capture())
	})

	t.Run("should display file content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))
		es := &fileeditor.State{
			Window:   w,
			File:     file,
			Content:  binding.NewString(),
			IsLoaded: binding.NewBool(),
			ErrorMsg: binding.NewString(),
			Bus:      mockBus,
		}

		// When
		res := texteditor.NewTextEditor(es)
		w.Canvas().SetContent(res)

		es.Content.Set("Hello world!") // nolint:errcheck
		es.IsLoaded.Set(true)          // nolint:errcheck

		// Then
		testutil.AssertImageMatches(t, "images/loaded-with-content.png", w.Canvas().Capture())
	})

	t.Run("should display error message", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))
		es := &fileeditor.State{
			Window:   w,
			File:     file,
			Content:  binding.NewString(),
			IsLoaded: binding.NewBool(),
			ErrorMsg: binding.NewString(),
			Bus:      mockBus,
		}

		// When
		res := texteditor.NewTextEditor(es)
		w.Canvas().SetContent(res)

		es.ErrorMsg.Set("Error loading file") // nolint:errcheck
		es.IsLoaded.Set(true)                 // nolint:errcheck

		// Then
		testutil.AssertImageMatches(t, "images/loaded-with-error.png", w.Canvas().Capture())
	})
}
