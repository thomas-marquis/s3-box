package uu_test

import (
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/tu"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
)

func TestTableBinding(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should initialize with zero dimensions by default", func(t *testing.T) {
		// Given & When
		table := uu.NewTableBinding(func(a, b string) bool { return a == b })

		// Then
		rows, cols := table.Dims()
		assert.Equal(t, 0, rows)
		assert.Equal(t, 0, cols)
	})

	t.Run("should initialize with given dimensions", func(t *testing.T) {
		// Given & When
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 3, 4)

		// Then
		rows, cols := table.Dims()
		assert.Equal(t, 3, rows)
		assert.Equal(t, 4, cols)
	})

	t.Run("should initialize with zero values for all cells when given dimensions", func(t *testing.T) {
		// Given & When
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)

		// Then - all cells should contain zero value for string
		val, err := table.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "", val)

		val, err = table.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, "", val)

		val, err = table.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "", val)

		val, err = table.ValueAt(1, 1)
		assert.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("should initialize with given dimensions for int type", func(t *testing.T) {
		// Given & When
		table := uu.NewTableBindingWithDim(func(a, b int) bool { return a == b }, 2, 2)

		// Then - all cells should contain zero value for int
		val, err := table.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, val)

		val, err = table.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, 0, val)
	})

	t.Run("should panic when initializing with negative rows", func(t *testing.T) {
		// Given & When & Then
		assert.Panics(t, func() {
			uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, -1, 2)
		})
	})

	t.Run("should panic when initializing with negative columns", func(t *testing.T) {
		// Given & When & Then
		assert.Panics(t, func() {
			uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, -1)
		})
	})
}

func TestTableBinding_Dims(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should return correct dimensions after initialization", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 5, 3)

		// When
		rows, cols := table.Dims()

		// Then
		assert.Equal(t, 5, rows)
		assert.Equal(t, 3, cols)
	})

	t.Run("should return updated dimensions after resize", func(t *testing.T) {
		// Given
		table := uu.NewTableBinding(func(a, b string) bool { return a == b })

		// When
		table.Resize(4, 4)
		rows, cols := table.Dims()

		// Then
		assert.Equal(t, 4, rows)
		assert.Equal(t, 4, cols)
	})
}

func TestTableBinding_Resize(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should resize to larger dimensions and fill new cells with zero values", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)
		assert.NoError(t, table.SetValue("a", 0, 0))
		assert.NoError(t, table.SetValue("b", 0, 1))
		assert.NoError(t, table.SetValue("c", 1, 0))
		assert.NoError(t, table.SetValue("d", 1, 1))

		// When
		table.Resize(4, 4)

		// Then
		rows, cols := table.Dims()
		assert.Equal(t, 4, rows)
		assert.Equal(t, 4, cols)

		// Original data should still be accessible
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "d", u.SkipV(table.ValueAt(1, 1)))

		// New cells should contain zero values
		assert.Equal(t, "", u.SkipV(table.ValueAt(0, 2)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(0, 3)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(1, 2)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(1, 3)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(2, 0)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(2, 1)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(2, 2)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(2, 3)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(3, 0)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(3, 1)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(3, 2)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(3, 3)))
	})

	t.Run("should resize to smaller dimensions", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 3, 3)
		assert.NoError(t, table.SetValue("a", 0, 0))
		assert.NoError(t, table.SetValue("b", 0, 1))
		assert.NoError(t, table.SetValue("c", 0, 2))
		assert.NoError(t, table.SetValue("d", 1, 0))
		assert.NoError(t, table.SetValue("e", 1, 1))
		assert.NoError(t, table.SetValue("f", 1, 2))
		assert.NoError(t, table.SetValue("g", 2, 0))
		assert.NoError(t, table.SetValue("h", 2, 1))
		assert.NoError(t, table.SetValue("i", 2, 2))

		// When
		table.Resize(2, 2)

		// Then
		rows, cols := table.Dims()
		assert.Equal(t, 2, rows)
		assert.Equal(t, 2, cols)

		// Only the first 2x2 cells should be accessible
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "d", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "e", u.SkipV(table.ValueAt(1, 1)))
	})

	t.Run("should resize to zero dimensions", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		assert.NoError(t, table.SetValue("a", 0, 0))
		assert.NoError(t, table.SetValue("b", 0, 1))

		// When
		table.Resize(0, 0)

		// Then
		rows, cols := table.Dims()
		assert.Equal(t, 0, rows)
		assert.Equal(t, 0, cols)
	})

	t.Run("should handle resize to same dimensions", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)
		assert.NoError(t, table.SetValue("a", 0, 0))
		assert.NoError(t, table.SetValue("b", 0, 1))
		assert.NoError(t, table.SetValue("c", 1, 0))
		assert.NoError(t, table.SetValue("d", 1, 1))

		// When
		originalRows, originalCols := table.Dims()
		table.Resize(originalRows, originalCols)

		// Then
		rows, cols := table.Dims()
		assert.Equal(t, originalRows, rows)
		assert.Equal(t, originalCols, cols)
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "d", u.SkipV(table.ValueAt(1, 1)))
	})

	t.Run("should fill new cells with zero value when resizing int table", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b int) bool { return a == b }, 1, 2)
		assert.NoError(t, table.SetValue(1, 0, 0))
		assert.NoError(t, table.SetValue(2, 0, 1))

		// When
		table.Resize(3, 3)

		// Then - new cells should contain zero value for int
		assert.Equal(t, 0, u.SkipV(table.ValueAt(0, 2)))
		assert.Equal(t, 0, u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, 0, u.SkipV(table.ValueAt(1, 1)))
		assert.Equal(t, 0, u.SkipV(table.ValueAt(1, 2)))
		assert.Equal(t, 0, u.SkipV(table.ValueAt(2, 0)))
		assert.Equal(t, 0, u.SkipV(table.ValueAt(2, 1)))
		assert.Equal(t, 0, u.SkipV(table.ValueAt(2, 2)))
	})

	t.Run("should fill new cells with zero value when resizing custom struct table", func(t *testing.T) {
		// Given
		type Person struct {
			Name string
			Age  int
		}
		comparator := func(a, b Person) bool {
			return a.Name == b.Name && a.Age == b.Age
		}
		table := uu.NewTableBindingWithDim(comparator, 1, 2)
		alice := Person{Name: "Alice", Age: 30}
		bob := Person{Name: "Bob", Age: 25}
		assert.NoError(t, table.SetValue(alice, 0, 0))
		assert.NoError(t, table.SetValue(bob, 0, 1))

		// When
		table.Resize(3, 3)

		// Then - new cells should contain zero value for Person
		val, err := table.ValueAt(0, 2)
		assert.NoError(t, err)
		assert.Equal(t, "", val.Name)
		assert.Equal(t, 0, val.Age)

		val, err = table.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "", val.Name)
		assert.Equal(t, 0, val.Age)
	})
}

func TestTableBinding_Set(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should fill entire table with new values", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)

		// When
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
		}
		err := table.Set(data)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "d", u.SkipV(table.ValueAt(1, 1)))
	})

	t.Run("should return error when data dimensions don't match table dimensions", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)

		// When - wrong number of rows
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
			{"e", "f"},
		}
		err := table.Set(data)

		// Then
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})

	t.Run("should return error when inner slice dimensions don't match", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)

		// When - wrong number of columns in first row
		data := [][]string{
			{"a", "b", "c"},
			{"d", "e"},
		}
		err := table.Set(data)

		// Then
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})

	t.Run("should return error when table is zero dimensions", func(t *testing.T) {
		// Given
		table := uu.NewTableBinding(func(a, b string) bool { return a == b })

		// When
		data := [][]string{
			{"a", "b"},
		}
		err := table.Set(data)

		// Then
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})

	t.Run("should work with integer type", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b int) bool { return a == b }, 2, 3)

		// When
		data := [][]int{
			{1, 2, 3},
			{4, 5, 6},
		}
		err := table.Set(data)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, 1, u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, 2, u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, 3, u.SkipV(table.ValueAt(0, 2)))
		assert.Equal(t, 4, u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, 5, u.SkipV(table.ValueAt(1, 1)))
		assert.Equal(t, 6, u.SkipV(table.ValueAt(1, 2)))
	})

	t.Run("should work with custom struct type", func(t *testing.T) {
		// Given
		type Person struct {
			Name string
			Age  int
		}
		comparator := func(a, b Person) bool {
			return a.Name == b.Name && a.Age == b.Age
		}
		table := uu.NewTableBindingWithDim(comparator, 2, 2)

		// When
		alice := Person{Name: "Alice", Age: 30}
		bob := Person{Name: "Bob", Age: 25}
		charlie := Person{Name: "Charlie", Age: 35}
		diana := Person{Name: "Diana", Age: 28}
		data := [][]Person{
			{alice, bob},
			{charlie, diana},
		}
		err := table.Set(data)

		// Then
		assert.NoError(t, err)
		val, err := table.ValueAt(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", val.Name)
		assert.Equal(t, 30, val.Age)

		val, err = table.ValueAt(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, "Bob", val.Name)
		assert.Equal(t, 25, val.Age)

		val, err = table.ValueAt(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, "Charlie", val.Name)
		assert.Equal(t, 35, val.Age)

		val, err = table.ValueAt(1, 1)
		assert.NoError(t, err)
		assert.Equal(t, "Diana", val.Name)
		assert.Equal(t, 28, val.Age)
	})

	t.Run("should return error when data is nil", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)

		// When
		err := table.Set(nil)

		// Then
		assert.Error(t, err)
	})

	t.Run("should work with empty table", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 0, 0)

		// When
		data := [][]string{}
		err := table.Set(data)

		// Then
		assert.NoError(t, err)
	})

	t.Run("should return error when inner slices have inconsistent lengths", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)

		// When - first row has 2 columns, second row has 3 columns
		data := [][]string{
			{"a", "b"},
			{"c", "d", "e"},
		}
		err := table.Set(data)

		// Then
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})
}

func TestTableBinding_ValueAt(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should return value at valid position", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 3)
		data := [][]string{
			{"a", "b", "c"},
		}
		assert.NoError(t, table.Set(data))

		// When & Then
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(0, 2)))
	})

	t.Run("should return error when accessing out of bounds", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		// When & Then
		_, err := table.ValueAt(5, 0)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)

		_, err = table.ValueAt(0, 5)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})
}

func TestTableBinding_SetValue(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should set value at valid position", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 3, 3)
		data := [][]string{
			{"a", "b", "c"},
			{"d", "e", "f"},
			{"g", "h", "i"},
		}
		assert.NoError(t, table.Set(data))

		// When
		assert.NoError(t, table.SetValue("A", 0, 0))
		assert.NoError(t, table.SetValue("E", 1, 1))
		assert.NoError(t, table.SetValue("I", 2, 2))

		// Then
		assert.Equal(t, "A", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(0, 2)))

		assert.Equal(t, "d", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "E", u.SkipV(table.ValueAt(1, 1)))
		assert.Equal(t, "f", u.SkipV(table.ValueAt(1, 2)))

		assert.Equal(t, "g", u.SkipV(table.ValueAt(2, 0)))
		assert.Equal(t, "h", u.SkipV(table.ValueAt(2, 1)))
		assert.Equal(t, "I", u.SkipV(table.ValueAt(2, 2)))
	})

	t.Run("should return error when setting value at out of bounds", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		// When & Then
		err := table.SetValue("x", 5, 0)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)

		err = table.SetValue("x", 0, 5)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})

	t.Run("should set value after resize to larger dimensions", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 1)
		data := [][]string{
			{"a"},
		}
		assert.NoError(t, table.Set(data))

		// When - resize first
		table.Resize(3, 3)
		// Then set values in new cells
		assert.NoError(t, table.SetValue("b", 0, 1))
		assert.NoError(t, table.SetValue("c", 1, 0))
		assert.NoError(t, table.SetValue("d", 2, 2))

		// Then
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "", u.SkipV(table.ValueAt(1, 1)))
		assert.Equal(t, "d", u.SkipV(table.ValueAt(2, 2)))
	})
}

func TestTableBinding_ItemAt(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should return item at valid position", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		// When
		item, err := table.ItemAt(0, 0)

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, item)
	})

	t.Run("should return error when item at out of bounds", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		// When & Then
		_, err := table.ItemAt(5, 0)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)

		_, err = table.ItemAt(0, 5)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)
	})

	t.Run("should return binding.Item that can be used to bind with a widget", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"initial", "value"},
		}
		assert.NoError(t, table.Set(data))

		e := widget.NewEntry()

		// When - bind the widget to the item returned by ItemAt
		item, err := table.ItemAt(0, 0)
		require.NoError(t, err)
		e.Bind(item)

		// Then - widget should display the value from the table
		assert.Equal(t, "initial", e.Text)

		// When - update via widget
		e.SetText("updated via widget")

		// Then - table should have the updated value
		assert.Equal(t, "updated via widget", u.SkipV(table.ValueAt(0, 0)))

		// When - update via table
		assert.NoError(t, table.SetValue("updated via table", 0, 0))

		// Then - widget should display the updated value
		assert.Equal(t, "updated via table", e.Text)
	})

	t.Run("should return same item instance when ItemAt is called twice with same coordinates", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
		}
		assert.NoError(t, table.Set(data))

		// When
		item1, err1 := table.ItemAt(0, 0)
		require.NoError(t, err1)

		item2, err2 := table.ItemAt(0, 0)
		require.NoError(t, err2)

		// Then
		assert.Same(t, item1, item2, "ItemAt should return the same cached instance for same coordinates")
	})

	t.Run("should trigger item listeners when table is resized and item position becomes invalid", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
		}
		require.NoError(t, table.Set(data))

		var itemListenerCallCount int32
		itemListener := binding.NewDataListener(func() {
			atomic.AddInt32(&itemListenerCallCount, 1)
		})

		item, err := table.ItemAt(1, 0)
		require.NoError(t, err)
		item.AddListener(itemListener)

		initialCount := atomic.LoadInt32(&itemListenerCallCount)
		require.Equal(t, int32(1), initialCount, "Listener should be triggered once when added")

		// When
		table.Resize(1, 2)

		// Then
		assert.Equal(t, int32(2), atomic.LoadInt32(&itemListenerCallCount),
			"Item listener should be triggered once more when table resize makes item position invalid")
	})

	t.Run("should not trigger item listeners unnecessarily during resize when position is still valid", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
		}
		require.NoError(t, table.Set(data))

		var itemListenerCallCount int32
		itemListener := binding.NewDataListener(func() {
			atomic.AddInt32(&itemListenerCallCount, 1)
		})

		item, err := table.ItemAt(0, 0)
		require.NoError(t, err)
		item.AddListener(itemListener)

		initialCount := atomic.LoadInt32(&itemListenerCallCount)
		require.Equal(t, int32(1), initialCount, "Listener should be triggered once when added")

		// When
		table.Resize(3, 3)

		// Then
		assert.Equal(t, int32(1), atomic.LoadInt32(&itemListenerCallCount),
			"Item listener should NOT be triggered again when position remains valid after resize")
	})
}

func TestTableBinding_Binding(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should allow removing listener", func(t *testing.T) {
		// Given
		table := uu.NewTableBinding(func(a, b string) bool { return a == b })

		var callCount int32
		listener := binding.NewDataListener(func() {
			atomic.AddInt32(&callCount, 1)
		})
		table.AddListener(listener)

		// When
		table.RemoveListener(listener)
		data := [][]string{
			{"a", "b"},
		}
		table.Resize(1, 2)
		_ = table.Set(data)

		// Then
		assert.Equal(t, int32(0), atomic.LoadInt32(&callCount))
	})

	t.Run("should notify data listener on value change", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		var callCount int32
		listener := binding.NewDataListener(func() {
			atomic.AddInt32(&callCount, 1)
		})
		table.AddListener(listener)

		// When
		assert.NoError(t, table.SetValue("new", 0, 0))

		// Then
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	})

	t.Run("should notify data listener on set", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)

		var callCount int32
		listener := binding.NewDataListener(func() {
			atomic.AddInt32(&callCount, 1)
		})
		table.AddListener(listener)

		// When
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		// Then
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	})

	t.Run("should notify data listener on resize", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 2)
		data := [][]string{
			{"a", "b"},
		}
		assert.NoError(t, table.Set(data))

		var callCount int32
		listener := binding.NewDataListener(func() {
			atomic.AddInt32(&callCount, 1)
		})
		table.AddListener(listener)

		// When
		table.Resize(2, 3)

		// Then
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	})
}

func TestTableBinding_WidgetBinding(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should bind each item to a widget", func(t *testing.T) {
		// Given
		e1 := widget.NewEntry()
		e2 := widget.NewEntry()
		e3 := widget.NewEntry()
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 3)
		data := [][]string{
			{"old a", "old b", "c"},
		}
		assert.NoError(t, table.Set(data))

		c := container.NewVBox(e1, e2, e3)

		w := fyne_test.NewWindow(c)
		w.Resize(fyne.NewSize(400, 300))
		canvas := w.Canvas()

		// When
		e1.Bind(u.SkipV(table.ItemAt(0, 0)))
		e2.Bind(u.SkipV(table.ItemAt(0, 1)))
		e3.Bind(u.SkipV(table.ItemAt(0, 2)))

		// Binding from entry to data
		e1.SetText("new a")

		// Binding from data to entry
		assert.NoError(t, table.SetValue("new b", 0, 1))

		// Then
		tu.AssertImageMatches(t, "images/table-binding-with-entries.png", canvas.Capture())
		assert.Equal(t, "new a", u.SkipV(table.ValueAt(0, 0)))
	})

	t.Run("should maintain binding consistency after resize", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 2, 2)
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
		}
		assert.NoError(t, table.Set(data))

		e1 := widget.NewEntry()
		e2 := widget.NewEntry()
		e3 := widget.NewEntry()
		e4 := widget.NewEntry()

		c := container.NewGridWithColumns(2, e1, e2, e3, e4)
		w := fyne_test.NewWindow(c)
		w.Resize(fyne.NewSize(400, 300))

		e1.Bind(u.SkipV(table.ItemAt(0, 0)))
		e2.Bind(u.SkipV(table.ItemAt(0, 1)))
		e3.Bind(u.SkipV(table.ItemAt(1, 0)))
		e4.Bind(u.SkipV(table.ItemAt(1, 1)))

		// When - resize larger
		table.Resize(3, 3)

		// Then - original bindings should still work
		assert.Equal(t, "a", e1.Text)
		assert.Equal(t, "b", e2.Text)
		assert.Equal(t, "c", e3.Text)
		assert.Equal(t, "d", e4.Text)
	})

	t.Run("should add row by resizing then setting values", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 1, 3)
		data := [][]string{
			{"a", "b", "c"},
		}
		require.NoError(t, table.Set(data))

		content := container.NewGridWithColumns(3)
		for j := range 3 {
			e := widget.NewEntry()
			e.Bind(u.SkipV(table.ItemAt(0, j)))
			content.Add(e)
		}

		w := fyne_test.NewWindow(content)
		w.Resize(fyne.NewSize(400, 300))
		canvas := w.Canvas()

		tu.AssertImageMatches(t, "images/table-binding-1-row.png", canvas.Capture())

		// When - add a new row by resizing
		table.Resize(2, 3)

		for j := range 3 {
			e := widget.NewEntry()
			e.Bind(u.SkipV(table.ItemAt(1, j)))
			content.Add(e)
		}

		tu.AssertImageMatches(t, "images/table-binding-2-rows-1-empty.png", canvas.Capture())

		assert.NoError(t, table.SetValue("d", 1, 0))
		assert.NoError(t, table.SetValue("e", 1, 1))
		assert.NoError(t, table.SetValue("f", 1, 2))

		// Then
		r, c := table.Dims()
		assert.Equal(t, 2, r)
		assert.Equal(t, 3, c)

		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "b", u.SkipV(table.ValueAt(0, 1)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(0, 2)))

		assert.Equal(t, "d", u.SkipV(table.ValueAt(1, 0)))
		assert.Equal(t, "e", u.SkipV(table.ValueAt(1, 1)))
		assert.Equal(t, "f", u.SkipV(table.ValueAt(1, 2)))

		tu.AssertImageMatches(t, "images/table-binding-2-rows.png", canvas.Capture())
	})

	t.Run("should detach widgets bound to removed row when downsizing", func(t *testing.T) {
		// Given
		table := uu.NewTableBindingWithDim(func(a, b string) bool { return a == b }, 3, 2)
		data := [][]string{
			{"a", "b"},
			{"c", "d"},
			{"e", "f"},
		}
		assert.NoError(t, table.Set(data))

		e1 := widget.NewEntry()
		e2 := widget.NewEntry()
		e3 := widget.NewEntry()

		c := container.NewVBox(e1, e2, e3)
		w := fyne_test.NewWindow(c)
		w.Resize(fyne.NewSize(400, 300))

		// Bind widgets to cells, including the last row
		e1.Bind(u.SkipV(table.ItemAt(0, 0)))
		e2.Bind(u.SkipV(table.ItemAt(1, 0)))
		e3.Bind(u.SkipV(table.ItemAt(2, 0)))

		// Sanity check
		assert.Equal(t, "e", e3.Text)

		// When - downsize to remove the last row
		table.Resize(2, 2)

		// Then - e3 should be detached
		// Verify by setting e3's text - it should not affect the table
		e3.SetText("detached value")

		// The table should still have original values
		assert.Equal(t, "a", u.SkipV(table.ValueAt(0, 0)))
		assert.Equal(t, "c", u.SkipV(table.ValueAt(1, 0)))

		// Accessing the removed row should now error
		_, err := table.ValueAt(2, 0)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)

		// Also verify that trying to get the item at the removed position errors
		_, err = table.ItemAt(2, 0)
		assert.Error(t, err)
		assert.Equal(t, uu.ErrTableOutOfBound, err)

		// e3 should retain its new value since it's detached
		// Note: This may pass even without explicit detachment if the binding item
		// returns errors when accessed out of bounds, preventing updates to the table
		assert.Equal(t, "detached value", e3.Text)
	})
}
