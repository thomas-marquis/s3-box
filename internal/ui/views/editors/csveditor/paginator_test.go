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

const csvContent16Rows = `id,name,age
0,Kate,22
1,Alice,30
2,Bob,25
3,Charlie,35
4,Pierre,40
5,Sarah,28
6,John,32
7,Gabriel,29
8,Eve,31
9,Lucas,27
10,Ursula,33
11,Victor,26
12,Wendy,34
13,Xavier,24
14,Yasmine,23
15,Zachary,21`

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
		assert.Equal(t, 1, p.PageNumber)
		assert.False(t, u.SkipV(p.HasHeader.Get()))
	})
}

func TestPaginator_MarksReady(t *testing.T) {
	t.Run("eager: should load all CSV data into table binding", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding(func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent16Rows)), false)

		// Then
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 10, rows, "Should have 10 rows (default page size)")
		assert.Equal(t, 3, cols, "Should have 3 columns")

		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val)

		val, err = tableBinding.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, "name", val)

		val, err = tableBinding.ValueAt(0, 2)
		assert.NoError(t, err)
		assert.Equal(t, "age", val)

		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "0", val)

		val, err = tableBinding.ValueAt(1, 1)
		assert.NoError(t, err)
		assert.Equal(t, "Kate", val)

		val, err = tableBinding.ValueAt(1, 2)
		assert.NoError(t, err)
		assert.Equal(t, "22", val)

		assert.True(t, p.HasNext(), "Should have next page")
	})

	t.Run("lazy: should load first page", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		tableBinding := uu.NewTableBinding(func(a, b string) bool { return a == b })
		p := csveditor.NewCsvPaginator(tableBinding)

		// When
		p.MarksReady(csv.NewReader(strings.NewReader(csvContent16Rows)), true)

		// Then
		rows, cols := tableBinding.Dims()
		assert.Equal(t, 10, rows, "Should have 10 rows (default page size)")
		assert.Equal(t, 3, cols, "Should have 3 columns")

		val, err := tableBinding.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "id", val)

		val, err = tableBinding.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, "name", val)

		val, err = tableBinding.ValueAt(0, 2)
		assert.NoError(t, err)
		assert.Equal(t, "age", val)

		val, err = tableBinding.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "0", val)

		val, err = tableBinding.ValueAt(1, 1)
		assert.NoError(t, err)
		assert.Equal(t, "Kate", val)

		val, err = tableBinding.ValueAt(1, 2)
		assert.NoError(t, err)
		assert.Equal(t, "22", val)

		assert.True(t, p.HasNext(), "Should have next page")
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
	fyne_test.NewApp()

	for _, tc := range []struct {
		name string
		lazy bool
	}{
		{"eager", false},
		{"lazy", true},
	} {
		t.Run(tc.name+": should return false when no more pages without header", func(t *testing.T) {
			// Given
			p := setupPaginator(t, 17)
			p.MarksReady(csv.NewReader(strings.NewReader(csvContent16Rows)), tc.lazy)

			// When
			res := p.Next()

			// Then
			assert.False(t, res, "Next should return true when more data is available")
			assert.Equal(t, 1, p.PageNumber)
		})

		t.Run(tc.name+": should return true then false when at end", func(t *testing.T) {
			// Given
			p := setupPaginator(t, 6)
			p.MarksReady(csv.NewReader(strings.NewReader(csvContent16Rows)), tc.lazy)

			// When & Then
			res := p.Next()
			assert.True(t, res, "should have a page 3")
			assert.Equal(t, 2, p.PageNumber)

			res = p.Next()
			assert.False(t, res, "should not have a page 4")
			assert.Equal(t, 3, p.PageNumber)
		})
	}

}

func TestPaginator_Prev(t *testing.T) {
	fyne_test.NewApp()

	for _, tc := range []struct {
		name string
		lazy bool
	}{
		{"eager", false},
		{"lazy", true},
	} {
		t.Run(tc.name+": should return false on first page", func(t *testing.T) {
			// Given
			p := setupPaginator(t, 1)
			csvContent := "a,b\nc,d"
			p.MarksReady(csv.NewReader(strings.NewReader(csvContent)), tc.lazy)

			// When
			res := p.Prev()

			// Then
			assert.False(t, res)
			assert.Equal(t, 1, p.PageNumber)
		})

		t.Run(tc.name+": should go to previous page", func(t *testing.T) {
			// Given
			p := setupPaginator(t, 6)
			p.MarksReady(csv.NewReader(strings.NewReader(csvContent16Rows)), tc.lazy)

			require.True(t, p.Next())
			require.Equal(t, 2, p.PageNumber)

			// When
			res := p.Prev()

			// Then
			// For eager mode: Prev should work and return false (no more previous pages)
			// For lazy mode: Prev is not supported, so it returns false and doesn't change page
			assert.False(t, res)
			if tc.lazy {
				// In lazy mode, Prev doesn't work, so page number stays the same
				assert.Equal(t, 2, p.PageNumber)
			} else {
				// In eager mode, Prev works, so page number changes
				assert.Equal(t, 1, p.PageNumber)
			}
		})
	}
}
