package texteditor

import (
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	fyne_widget "fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/fileeditor"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	"go.uber.org/mock/gomock"
)

// uv run ./tools/diff_images.py --folders internal/ui/views/editors/texteditor/testdata/images internal/ui/views/editors/texteditor/testdata/failed/images --color "red"

func TestFileEditor_saving(t *testing.T) {
	fyne_test.NewApp()

	lastModified := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.txt", rootDir,
		directory.WithFileSize(1024),
		directory.WithFileLastModified(lastModified),
	)

	t.Run("should save updated content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).AnyTimes()

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

		res := NewTextEditor(es)
		canvas := w.Canvas()
		canvas.SetContent(res)

		// When & Then
		fyne_test.Type(res.textEditor, "my new content")
		fyne_test.Tap(res.saveBtn.ToolbarObject().(*fyne_widget.Button))
		testutil.AssertImageMatches(t, "images/updated-and-saving.png", canvas.Capture())

		events <- fileeditor.NewSaveSuccessEvent(file, "my new content")
		testutil.AssertImageMatches(t, "images/updated-and-saved.png", canvas.Capture())
	})

	t.Run("should display save error", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)
		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)
		mockBus.EXPECT().Publish(gomock.Any()).AnyTimes()

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

		res := NewTextEditor(es)
		canvas := w.Canvas()
		canvas.SetContent(res)

		// When & Then
		fyne_test.Type(res.textEditor, "my new content")
		fyne_test.Tap(res.saveBtn.ToolbarObject().(*fyne_widget.Button))

		events <- fileeditor.NewSaveFailureEvent(file, errors.New("failed to save"))
		testutil.AssertImageMatches(t, "images/saving-error.png", canvas.Capture())
	})
}
