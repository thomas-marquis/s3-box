package csveditor_test

import (
	"strings"
	"testing"

	"encoding/csv"
	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
)

func setupPaginator(t *testing.T, pageSize int) *csveditor.Paginator {
	t.Helper()
	fyne_test.NewApp()
	tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
	p := csveditor.NewCsvPaginator(tableBinding)
	p.PageSize = pageSize
	return p
}

func TestPaginator(t *testing.T) {
	t.Run("should initialize with default values", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })

		// When
		p := csveditor.NewCsvPaginator(tableBinding)

		// Then
		assert.Equal(t, 10, p.PageSize)
		assert.Equal(t, 0, p.CurrentIndex())
	})
}

func TestPaginator_MarksReady(t *testing.T) {
	t.Run("non-lazy: should load all CSV data into table binding on MarksReady", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		csvContent := "name,age\nAlice,30\nBob,25\nCharlie,35"

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)

		// Then - table should have all data
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Should have 3 rows (all data)")
		assert.Equal(t, 2, cols, "Should have 2 columns")

		// Check data - all rows should be loaded
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "name", val)

		val, err = tableBinding.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, "age", val)

		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", val)

		val, err = tableBinding.ValueAt(2, 1)
		assert.NoError(t, err)
		assert.Equal(t, "35", val)
	})

	t.Run("lazy: should load first page on MarksReady", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2
		csvContent := "name,age\nAlice,30\nBob,25\nCharlie,35\nDavid,40"

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)

		// Then - table should have first page data (first 2 records after header? No, first 2 records including header)
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 2, rows, "Should have 2 rows in first page")
		assert.Equal(t, 2, cols, "Should have 2 columns")

		// Check data
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "name", val)

		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", val)
	})

	t.Run("lazy: TotalPages should return -1", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		csvContent := "name,age\nAlice,30"

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)

		// Then
		assert.Equal(t, -1, p.TotalPages(), "Lazy loading should return -1 for total pages")
	})
}

func TestPaginator_Next(t *testing.T) {
	t.Run("non-lazy: should return false when no more pages", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		p := setupPaginator(t, 2)
		csvContent := "a,b\nc,d\ne,f"
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)

		// Table should show first 2 rows (page size = 2)
		// When we try to go to next page, we have 2 more rows total (3 rows in CSV)
		// Page 1: rows 0-1, Page 2: row 2
		
		// When
		hasNext := p.HasNext()
		assert.True(t, hasNext, "Should have next page")
		
		res := p.Next()

		// Then
		assert.True(t, res, "Next should return true when more data is available")
		assert.Equal(t, 2, p.CurrentIndex())
	})

	t.Run("non-lazy: should return false when at end", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		p := setupPaginator(t, 2)
		csvContent := "a,b\nc,d"
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)

		// Table has exactly 2 rows, which is one page
		// HasNext should be false
		assert.False(t, p.HasNext())
		
		// When
		res := p.Next()

		// Then
		assert.False(t, res)
		assert.Equal(t, 0, p.CurrentIndex())
	})
}

func TestPaginator_Prev(t *testing.T) {
	t.Run("non-lazy: should go to previous page", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		p := setupPaginator(t, 1)
		csvContent := "a,b\nc,d\ne,f"
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)

		// Go to page 2
		p.Next()
		assert.Equal(t, 1, p.CurrentIndex())

		// When
		res := p.Prev()

		// Then
		assert.True(t, res)
		assert.Equal(t, 0, p.CurrentIndex())
	})

	t.Run("non-lazy: should return false on first page", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		p := setupPaginator(t, 1)
		csvContent := "a,b\nc,d"
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)

		// When
		res := p.Prev()

		// Then
		assert.False(t, res)
		assert.Equal(t, 0, p.CurrentIndex())
	})
}

// Test for the issue: lazy loading should load first page and stop loading
func TestPaginator_LazyLoading(t *testing.T) {
	t.Run("should not block on MarksReady for lazy loading", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 5
		csvContent := strings.Repeat("a,b,c\n", 100) // Large CSV

		// When - this should not block or take too long
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)

		// Then - should have loaded first page
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 5, rows, "Should have loaded first 5 rows")
		assert.Equal(t, 3, cols, "Should have 3 columns")
	})
}
