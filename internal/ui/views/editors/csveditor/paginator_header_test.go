package csveditor_test

import (
	"encoding/csv"
	"strings"
	"testing"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
)

// Test cases for header row handling in paginator

func TestPaginator_HeaderHandling_Eager(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should display header row when HasHeader is true on page 1", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25
3,Charlie,35`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(true))

		// Then - table should include header + 2 data rows = 3 rows
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Should have header + 2 data rows")
		assert.Equal(t, 3, cols, "Should have 3 columns")

		// Check header row
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val)

		val, err = tableBinding.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, "name", val)

		// Check first data row
		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "1", val)

		val, err = tableBinding.ValueAt(1, 1)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", val)

		// Check second data row
		val, err = tableBinding.ValueAt(2, 0)
		assert.NoError(t, err)
		assert.Equal(t, "2", val)
	})

	t.Run("should NOT display header row when HasHeader is false on page 1", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25
3,Charlie,35`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(false))

		// Then - table should have only 2 data rows (no header)
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 2, rows, "Should have only 2 data rows")
		assert.Equal(t, 3, cols, "Should have 3 columns")

		// First row should be data, not header
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val, "First row should be the CSV first line (treated as data)")

		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "1", val)
	})

	t.Run("should display header on all pages when HasHeader is true", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25
3,Charlie,35
4,David,40`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(true))

		// Then - page 1 should have header + 2 data rows
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Page 1: header + 2 data rows")

		// When - go to page 2
		p.Next()

		// Then - page 2 should also have header + 2 data rows
		rows, cols = tableBinding.Dims()
		assert.Equal(t, 3, rows, "Page 2: header + 2 data rows")

		// Check header is still there
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val)

		// Check data rows are correct (rows 3 and 4 from CSV, which are indices 3 and 4 in allRecords)
		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "3", val)

		val, err = tableBinding.ValueAt(2, 0)
		assert.NoError(t, err)
		assert.Equal(t, "4", val)
	})

	t.Run("should NOT include header in page count", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25
3,Charlie,35
4,David,40
5,Eve,28`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(true))

		// Then - should have 3 pages (5 data rows / 2 per page, header doesn't count)
		// 5 data rows, page size 2 -> ceil(5/2) = 3 pages
		assert.Equal(t, 3, p.TotalPages(), "Should have 3 pages for 5 data rows with page size 2")
	})
}

func TestPaginator_HeaderHandling_Lazy(t *testing.T) {
	fyne_test.NewApp()

	t.Run("lazy: should display header row when HasHeader is true on page 1", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25
3,Charlie,35
4,David,40
5,Eve,28`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)
		u.Skip(p.HasHeader.Set(true))

		// Then - table should include header + 2 data rows = 3 rows
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Lazy: Should have header + 2 data rows on page 1")
		assert.Equal(t, 3, cols, "Should have 3 columns")

		// Check header row
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val)

		// Check first data row
		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "1", val)

		// Check second data row
		val, err = tableBinding.ValueAt(2, 0)
		assert.NoError(t, err)
		assert.Equal(t, "2", val)
	})

	t.Run("lazy: should display header on page 2", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25
3,Charlie,35
4,David,40
5,Eve,28`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)
		u.Skip(p.HasHeader.Set(true))

		// Go to page 2
		p.Next()

		// Then - table should still include header + 2 data rows
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Lazy page 2: Should have header + 2 data rows")

		// Check header is still there
		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val)

		// Check data rows (should be rows 3 and 4 from CSV)
		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "3", val)

		val, err = tableBinding.ValueAt(2, 0)
		assert.NoError(t, err)
		assert.Equal(t, "4", val)
	})

	t.Run("lazy: TotalPages should return -1", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)

		// Then
		assert.Equal(t, -1, p.TotalPages(), "Lazy loading should return -1 for total pages")
	})
}

func TestPaginator_HeaderToggle(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should update table when HasHeader is toggled", func(t *testing.T) {
		// Given
		csvContent := `id,name,age
1,Alice,30
2,Bob,25`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When - load with header enabled
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(true))

		// Then - should have header + 2 data rows
		rows, _ := tableBinding.Dims()
		assert.Equal(t, 3, rows)

		val, _ := tableBinding.ValueAt(0, 0)
		assert.Equal(t, "id", val)

		// When - disable header
		u.Skip(p.HasHeader.Set(false))

		// Then - should have only 2 data rows (no header)
		rows, _ = tableBinding.Dims()
		assert.Equal(t, 2, rows)

		// First row should now be the CSV header (treated as data)
		val, _ = tableBinding.ValueAt(0, 0)
		assert.Equal(t, "id", val)

		// When - re-enable header
		u.Skip(p.HasHeader.Set(true))

		// Then - should have header + 2 data rows again
		rows, _ = tableBinding.Dims()
		assert.Equal(t, 3, rows)

		val, _ = tableBinding.ValueAt(0, 0)
		assert.Equal(t, "id", val)
	})
}

func TestPaginator_EmptyCSV(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should handle empty CSV", func(t *testing.T) {
		// Given
		csvContent := ""
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)

		// Then
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 0, rows)
		assert.Equal(t, 0, cols)
	})

	t.Run("should handle CSV with only header", func(t *testing.T) {
		// Given
		csvContent := "id,name,age"
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(true))

		// Then - should have only header row
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 1, rows)
		assert.Equal(t, 3, cols)

		val, _ := tableBinding.ValueAt(0, 0)
		assert.Equal(t, "id", val)
	})

	t.Run("should handle CSV with only header and HasHeader false", func(t *testing.T) {
		// Given
		csvContent := "id,name,age"
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 2

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(false))

		// Then - should have 1 data row (the header line treated as data)
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 1, rows)
		assert.Equal(t, 3, cols)

		val, _ := tableBinding.ValueAt(0, 0)
		assert.Equal(t, "id", val)
	})
}

func TestPaginator_SmallFileFitsInOnePage(t *testing.T) {
	fyne_test.NewApp()

	t.Run("eager: small file with header should fit in one page", func(t *testing.T) {
		// Given
		csvContent := `id,name
1,Alice
2,Bob`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 10 // Large page size

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), false)
		u.Skip(p.HasHeader.Set(true))

		// Then - all data should fit in one page
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Should have header + 2 data rows")
		assert.Equal(t, 2, cols)

		// Should not have next page
		assert.False(t, p.HasNext())
		assert.Equal(t, 1, p.TotalPages())
	})

	t.Run("lazy: small file should load all data in first page", func(t *testing.T) {
		// Given
		csvContent := `id,name
1,Alice
2,Bob`
		tableBinding := uu.NewTableBinding[string](func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)
		p.PageSize = 10

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), true)
		u.Skip(p.HasHeader.Set(true))

		// Then
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 3, rows, "Lazy: should have header + 2 data rows")
		assert.Equal(t, 2, cols)

		// Total pages should be -1 (unknown)
		assert.Equal(t, -1, p.TotalPages())
	})
}
