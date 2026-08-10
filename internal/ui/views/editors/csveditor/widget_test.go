package csveditor_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	csvContent = `id,name,age
1,toto,12
2,lolo,13`
)

func TestCsvEditorWidget(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image matching tests in short mode")
	}

	var file *directory.File
	testutil.MakeDirectory(t, "",
		testutil.AsRoot(),
		testutil.WithFiles("data.csv"),
		testutil.FileTo("data.csv", &file))

	t.Run("should display the csv content when loaded successfully", func(t *testing.T) {
		// Given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		bus := inmemory.NewBus(ctx)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))

		ed := csveditor.New(bus, w, file)
		mockContent := &directory.InMemoryContent{
			Data: []byte(csvContent),
		}

		// When
		res := ed.CreateWidget()
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: mockContent,
		}))
		w.SetContent(res)

		// Then
		testutil.AssertImageMatches(t, "images/loaded-successfully.png", w.Canvas().Capture())
	})
}
