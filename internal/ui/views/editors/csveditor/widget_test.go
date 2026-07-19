package csveditor_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	"go.uber.org/mock/gomock"
)

var (
	lastModified = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
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
		ctrl := gomock.NewController(t)
		mockBus := mocks_event.NewMockBus(ctrl)

		w := fyne_test.NewWindow(nil)
		w.Resize(fyne.NewSize(500, 300))

		ed := csveditor.New(mockBus, w, file)
		mockContent := &directory.InMemoryContent{
			Data: []byte(csvContent),
		}

		// When
		res := ed.CreateWidget()
		ed.OnLoaded(mockContent, nil)
		w.SetContent(res)

		// Then
		testutil.AssertImageMatches(t, "images/loaded-successfully.png", w.Canvas().Capture())
	})
}
