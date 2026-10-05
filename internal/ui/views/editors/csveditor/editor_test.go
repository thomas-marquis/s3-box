package csveditor_test

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/docker/docker/daemon/events"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
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
	ed.UpdatePageLabel()

	// Then - Page 1
	page, _ := ed.PageLabel.Get()
	assert.Equal(t, "1 / 2", page)
	records, _ := ed.Records.Get()
	assert.Len(t, records, 2)
	assert.Equal(t, []string{"1", "a"}, records[0])

	// When - Next Page
	ed.NextPage()

	// Then - Page 2
	page, _ = ed.PageLabel.Get()
	assert.Equal(t, "2 / 2", page)
	records, _ = ed.Records.Get()
	assert.Len(t, records, 1)
	assert.Equal(t, []string{"3", "c"}, records[0])

	// When - Prev Page
	ed.PrevPage()

	// Then - Back to Page 1
	page, _ = ed.PageLabel.Get()
	assert.Equal(t, "1 / 2", page)
	records, _ = ed.Records.Get()
	assert.Len(t, records, 2)
	assert.Equal(t, []string{"1", "a"}, records[0])

	content := ed.GetContent()
	assert.Equal(t, `1,a
2,b
3,c`, content)
}

func TestCsvEditor_TableBinding_NonLazy(t *testing.T) {
	test.NewApp()
	bus := inmemory.NewBus(context.Background())
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.csv", rootDir)

	edInterface := csveditor.New(bus, test.NewWindow(nil), file)
	ed := edInterface.(*csveditor.Editor)
	ed.Paginator.PageSize = 2

	csvContent := `id,name,age
1,toto,12
2,lolo,13
3,alice,25`

	mockContent := &directory.InMemoryContent{
		Data: []byte(csvContent),
	}

	// When - Load the file
	bus.Publish(event.New(editor.Loaded{
		Editor:  ed,
		Content: mockContent,
	}))

	// Then - the table should have been initialized with all data
	rows, cols := ed.TableBinding.Dims()
	assert.Equal(t, 3, rows, "Should have 3 rows (including header)")
	assert.Equal(t, 3, cols, "Should have 3 columns")

	// Check some cell values
	val, err := ed.TableBinding.ValueAt(0, 0)
	assert.NoError(t, err)
	assert.Equal(t, "id", val)

	val, err = ed.TableBinding.ValueAt(1, 1)
	assert.NoError(t, err)
	assert.Equal(t, "toto", val)

	val, err = ed.TableBinding.ValueAt(2, 2)
	assert.NoError(t, err)
	assert.Equal(t, "13", val)
}

func TestCsvEditor_TableBinding_Lazy(t *testing.T) {
	test.NewApp()
	bus := inmemory.NewBus(context.Background())
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.csv", rootDir)

	edInterface := csveditor.NewLazy(bus, test.NewWindow(nil), file)
	ed := edInterface.(*csveditor.Editor)
	ed.Paginator.PageSize = 2

	csvContent := `id,name,age
1,toto,12
2,lolo,13
3,alice,25
4,bob,31`

	mockContent := &directory.InMemoryContent{
		Data: []byte(csvContent),
	}

	// When - Load the file (lazy)
	bus.Publish(event.New(editor.Loaded{
		Editor:  ed,
		Content: mockContent,
	}))

	// Then - the table should be initialized but empty (lazy loading)
	rows, cols := ed.TableBinding.Dims()
	assert.Equal(t, 0, rows, "Lazy: table should start with 0 rows")
	assert.Equal(t, 0, cols, "Lazy: table should start with 0 cols")

	// When - we don't explicitly load page 1, but the paginator should handle it
	// The table should be filled with the first page of data
	// For now, we need to check the implementation to see how lazy loading works
}

func TestCsvEditor_TableBinding_Pagination(t *testing.T) {
	test.NewApp()
	bus := inmemory.NewBus(context.Background())
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.csv", rootDir)

	edInterface := csveditor.New(bus, test.NewWindow(nil), file)
	ed := edInterface.(*csveditor.Editor)
	ed.Paginator.PageSize = 2

	csvContent := `id,name,age
1,toto,12
2,lolo,13
3,alice,25
4,bob,31`

	mockContent := &directory.InMemoryContent{
		Data: []byte(csvContent),
	}

	// When - Load the file
	bus.Publish(event.New(editor.Loaded{
		Editor:  ed,
		Content: mockContent,
	}))

	// Then - page 1 should be displayed
	rows, cols := ed.TableBinding.Dims()
	assert.Equal(t, 3, rows, "Full load: should have all 3 data rows + header")
	assert.Equal(t, 3, cols)

	// Check page label
	page, _ := ed.PageLabel.Get()
	assert.Equal(t, "1 / 2", page, "Should show page 1 of 2")

	// When - go to page 2
	ed.NextPage()

	// Then - page 2 should be displayed
	// In non-lazy mode with full table, pagination should still work
	// The table still contains all data, but the paginator controls which rows are visible
	page, _ = ed.PageLabel.Get()
	assert.Equal(t, "2 / 2", page, "Should show page 2 of 2")
}

func TestCsvEditor_TableBinding_LazyPagination(t *testing.T) {
	test.NewApp()
	bus := inmemory.NewBus(context.Background())
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test-lazy.csv", rootDir)

	edInterface := csveditor.NewLazy(bus, test.NewWindow(nil), file)
	ed := edInterface.(*csveditor.Editor)
	ed.Paginator.PageSize = 2

	csvContent := `id,name,age
1,toto,12
2,lolo,13
3,alice,25
4,bob,31
5,charlie,28`

	mockContent := &directory.InMemoryContent{
		Data: []byte(csvContent),
	}

	// When - Load the file (lazy)
	bus.Publish(event.New(editor.Loaded{
		Editor:  ed,
		Content: mockContent,
	}))

	// Then - page label should show "?" for total pages
	page, _ := ed.PageLabel.Get()
	assert.True(t, strings.Contains(page, "?"), "Lazy mode: page label should contain '?' for unknown total pages")

	// When - go to page 2
	ed.NextPage()

	// Then - page label should update
	page, _ = ed.PageLabel.Get()
	assert.True(t, strings.Contains(page, "?"), "Lazy mode: page label should still contain '?'")
}

func TestCsvEditor_TableBinding_HeaderVisibility(t *testing.T) {
	test.NewApp()
	bus := inmemory.NewBus(context.Background())
	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
	file, _ := directory.NewFile("test.csv", rootDir)

	edInterface := csveditor.New(bus, test.NewWindow(nil), file)
	ed := edInterface.(*csveditor.Editor)

	csvContent := `id,name,age
1,toto,12
2,lolo,13`

	mockContent := &directory.InMemoryContent{
		Data: []byte(csvContent),
	}

	// When - Load the file
	bus.Publish(events.New(editor.Loaded{
		Editor:  ed,
		Content: mockContent,
	}))

	// Then - table should have 3 rows (including header)
	rows, cols := ed.TableBinding.Dims()
	assert.Equal(t, 3, rows)
	assert.Equal(t, 3, cols)

	// When - header is enabled
	ed.Paginator.HasHeader.Set(true)

	// Then - table should still have the same dimensions
	// The header visibility is managed by the widget, not by resizing the table
	rows, cols = ed.TableBinding.Dims()
	assert.Equal(t, 3, rows)
	assert.Equal(t, 3, cols)
}
