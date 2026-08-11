package csveditor_test

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
)

type paginatorFixture struct {
	t       *testing.T
	Binding binding.List[[]string]
	P       *csveditor.Paginator
}

func setupPaginator(t *testing.T, pageSize int) *paginatorFixture {
	t.Helper()
	fyne_test.NewApp()
	bound := binding.NewList[[]string](slices.Equal)
	p := csveditor.NewCsvPaginator(bound)
	p.PageSize = pageSize

	return &paginatorFixture{
		t:       t,
		Binding: bound,
		P:       p,
	}
}

func TestPaginator(t *testing.T) {
	t.Run("should initialize with default values", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		bound := binding.NewList[[]string](slices.Equal)

		// When
		p := csveditor.NewCsvPaginator(bound)

		// Then
		assert.Equal(t, 10, p.PageSize)
		assert.Equal(t, 0, p.CurrentIndex)
		assert.Empty(t, p.Records)
	})
}

func TestPaginator_Append(t *testing.T) {
	t.Run("should add records and populate first page", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 2)

		// When
		fxt.P.Append([]string{"row1", "col2"})
		fxt.P.Append([]string{"row2", "col2"})

		// Then
		assert.Len(t, fxt.P.Records, 2)
		assert.Equal(t, csveditor.Record{"row1", "col2"}, fxt.P.Records[0])

		records, _ := fxt.Binding.Get()
		require.Len(t, records, 2)
		assert.Equal(t, []string{"row1", "col2"}, records[0])
		assert.Equal(t, []string{"row2", "col2"}, records[1])
	})

	t.Run("should update bound list gracefully when not on first page", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 2)
		fxt.P.Append([]string{"1", "a"})
		fxt.P.Append([]string{"2", "b"})
		fxt.P.Append([]string{"3", "c"})
		fxt.P.Next() // Should be on second page showing ["3", "c"]

		// When
		fxt.P.Append([]string{"4", "d"})

		// Then
		assert.Equal(t, 2, fxt.P.CurrentIndex)
		records, _ := fxt.Binding.Get()
		require.Len(t, records, 2)
		assert.Equal(t, []string{"3", "c"}, records[0])
		assert.Equal(t, []string{"4", "d"}, records[1])
	})
}

func TestPaginator_Next(t *testing.T) {
	t.Run("should return false and not increment when on the last page", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 2)
		fxt.P.Append([]string{"1", "a"})
		fxt.P.Append([]string{"2", "b"})

		// When
		res := fxt.P.Next()

		// Then
		assert.False(t, res, "Next should return false when no more pages")
		assert.Equal(t, 0, fxt.P.CurrentIndex, "CurrentIndex should not have changed")
	})

	t.Run("should return false when the last page is reached", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 2)
		fxt.P.Append([]string{"1", "a"})
		fxt.P.Append([]string{"2", "b"})
		fxt.P.Append([]string{"3", "c"})
		fxt.P.Append([]string{"4", "d"})

		// When
		res := fxt.P.Next()

		// Then
		assert.False(t, res, "Next should return false on the last page")
		assert.Equal(t, 2, fxt.P.CurrentIndex, "CurrentIndex should not have changed")
	})

	t.Run("should update the bound list and increment index", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 1)
		fxt.P.Append([]string{"1", "a"})
		fxt.P.Append([]string{"2", "b"})

		// When
		res := fxt.P.Next()

		// Then
		assert.False(t, res)
		assert.Equal(t, 1, fxt.P.CurrentIndex)

		records, _ := fxt.Binding.Get()
		require.Len(t, records, 1)
		assert.Equal(t, []string{"2", "b"}, records[0])
	})
}

func TestPaginator_Prev(t *testing.T) {
	t.Run("should return false and not decrement when on the first page", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 2)
		fxt.P.Append([]string{"1", "a"})
		fxt.P.Append([]string{"2", "b"})

		// When
		res := fxt.P.Prev()

		// Then
		assert.False(t, res, "Prev should return false when on the first page")
		assert.Equal(t, 0, fxt.P.CurrentIndex, "CurrentIndex should not have changed")
	})

	t.Run("should update the bound list and decrement index", func(t *testing.T) {
		// Given
		fxt := setupPaginator(t, 1)
		fxt.P.Append([]string{"1", "a"})
		fxt.P.Append([]string{"2", "b"})
		fxt.P.Next() // Go to second page

		// When
		res := fxt.P.Prev()

		// Then
		assert.True(t, res)
		assert.Equal(t, 0, fxt.P.CurrentIndex)

		records, _ := fxt.Binding.Get()
		require.Len(t, records, 1)
		assert.Equal(t, []string{"1", "a"}, records[0])
	})
}
