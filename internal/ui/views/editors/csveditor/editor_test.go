package csveditor_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
)

func TestCsvEditor_Pagination(t *testing.T) {
	test.NewApp()
	bus := inmemory.NewBus(context.Background())
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.csv", rootDir)

	edInterface := csveditor.New(bus, test.NewWindow(nil), file)
	ed := edInterface.(*csveditor.Editor)
	ed.Paginator.PageSize = 2

	// Given
	ed.Paginator.Append([]string{"1", "a"})
	ed.Paginator.Append([]string{"2", "b"})
	ed.Paginator.Append([]string{"3", "c"})

	// Then - Page 1
	records, _ := ed.Records.Get()
	assert.Len(t, records, 2)
	assert.Equal(t, []string{"1", "a"}, records[0])

	// When - Next Page
	ed.NextPage()

	// Then - Page 2
	records, _ = ed.Records.Get()
	assert.Len(t, records, 1)
	assert.Equal(t, []string{"3", "c"}, records[0])

	// When - Prev Page
	ed.PrevPage()

	// Then - Back to Page 1
	records, _ = ed.Records.Get()
	assert.Len(t, records, 2)
	assert.Equal(t, []string{"1", "a"}, records[0])

	content := ed.GetContent()
	assert.Equal(t, `1,a
2,b
3,c`, content)
}
