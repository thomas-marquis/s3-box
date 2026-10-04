package uu

import (
	"errors"
	"sync"

	"fyne.io/fyne/v2/data/binding"
)

var (
	ErrTableOutOfBound = errors.New("the specified coordinates are out of bound of the existing table")
)

type TableBinding[T any] struct {
	comparator     func(T, T) bool
	data           []T
	nbRows, nbCols int
	listeners      []binding.DataListener
	items          map[[2]int]*positionTrackingItem[T]
	lock           sync.RWMutex
}

// NewTableBinding constructs a binding object that holds a 2-dimension data table.
func NewTableBinding[T any](comparator func(T, T) bool) *TableBinding[T] {
	return NewTableBindingWithDim(comparator, 0, 0)
}

// NewTableBindingWithDim constructs a binding object that holds a 2-dimension data table
// with the specified initial dimensions.
func NewTableBindingWithDim[T any](comparator func(T, T) bool, rows, cols int) *TableBinding[T] {
	if rows < 0 || cols < 0 {
		panic("TableBinding dimensions cannot be negative")
	}

	data := make([]T, rows*cols)

	return &TableBinding[T]{
		comparator: comparator,
		data:       data,
		nbRows:     rows,
		nbCols:     cols,
		listeners:  make([]binding.DataListener, 0),
		items:      make(map[[2]int]*positionTrackingItem[T]),
	}
}

func (b *TableBinding[T]) AddListener(dl binding.DataListener) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.listeners = append(b.listeners, dl)
}

func (b *TableBinding[T]) RemoveListener(dl binding.DataListener) {
	b.lock.Lock()
	defer b.lock.Unlock()
	for i, l := range b.listeners {
		if l == dl {
			b.listeners = append(b.listeners[:i], b.listeners[i+1:]...)
			return
		}
	}
}

func (b *TableBinding[T]) trigger() {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, l := range b.listeners {
		l.DataChanged()
	}
}

func (b *TableBinding[T]) Dims() (rows, cols int) {
	return b.nbRows, b.nbCols
}

// Resize resizes the table to the specified dimensions.
// If the new size is larger than the current size, new cells are filled with zero values.
// If the new size is smaller, data is truncated.
// Existing data is preserved at the same (row, col) positions.
// Note: After resize, any previously returned binding.Items from ItemAt are invalidated
// and should not be used. New items should be obtained via ItemAt after resize.
func (b *TableBinding[T]) Resize(rows, cols int) {
	if rows < 0 || cols < 0 {
		return
	}

	b.lock.Lock()

	oldRows, oldCols := b.nbRows, b.nbCols

	// Before resizing, check which cached items will become invalid
	// and collect them to trigger their listeners later
	invalidItems := make([]*positionTrackingItem[T], 0)
	for key, item := range b.items {
		row, col := key[0], key[1]
		// Check if this position will be invalid after resize
		if row >= rows || col >= cols {
			// Collect invalid items
			invalidItems = append(invalidItems, item)
			// Remove from cache
			delete(b.items, key)
			// Remove from listeners list
			for i, l := range b.listeners {
				if l == item {
					b.listeners = append(b.listeners[:i], b.listeners[i+1:]...)
					break
				}
			}
		}
	}

	totalCells := rows * cols

	// Create new data slice with zero values
	newData := make([]T, totalCells)

	// Copy existing data preserving 2D positions
	for i := 0; i < rows && i < oldRows; i++ {
		for j := 0; j < cols && j < oldCols; j++ {
			newIdx := b.toIndexWithDims(i, j, cols)
			oldIdx := b.toIndexWithDims(i, j, oldCols)
			if oldIdx < len(b.data) {
				newData[newIdx] = b.data[oldIdx]
			}
		}
	}

	b.data = newData
	b.nbRows = rows
	b.nbCols = cols

	// Unlock before triggering listeners to avoid deadlocks
	b.lock.Unlock()

	// Trigger listeners for invalid items
	for _, item := range invalidItems {
		item.trigger()
	}

	// Trigger notifications for all remaining listeners
	b.trigger()
}

// Set fills the entire table with the provided 2D slice.
// The provided data must have the exact same dimensions as the table, otherwise an error is returned.
func (b *TableBinding[T]) Set(data [][]T) error {
	if len(data) != b.nbRows {
		return ErrTableOutOfBound
	}

	// Check all rows have consistent column count
	for i := range data {
		if len(data[i]) != b.nbCols {
			return ErrTableOutOfBound
		}
	}

	// Flatten the 2D data
	flatData := make([]T, b.nbRows*b.nbCols)
	for i := range data {
		for j := range data[i] {
			flatData[b.toIndex(i, j)] = data[i][j]
		}
	}

	b.lock.Lock()
	b.data = flatData
	b.lock.Unlock()

	// Trigger all cached items' listeners
	b.lock.RLock()
	for _, item := range b.items {
		item.trigger()
	}
	b.lock.RUnlock()

	// Trigger table-level listeners
	b.trigger()

	return nil
}

func (b *TableBinding[T]) SetValue(val T, row, col int) error {
	if row >= b.nbRows || col >= b.nbCols {
		return ErrTableOutOfBound
	}

	b.lock.Lock()
	idx := b.toIndex(row, col)
	b.data[idx] = val
	b.lock.Unlock()

	// Trigger the item's listeners for this specific cell if it exists in cache
	key := [2]int{row, col}
	b.lock.RLock()
	if item, exists := b.items[key]; exists {
		b.lock.RUnlock()
		item.trigger()
	} else {
		b.lock.RUnlock()
	}

	// Trigger table-level listeners
	b.trigger()

	return nil
}

func (b *TableBinding[T]) ValueAt(row, col int) (val T, err error) {
	if row >= b.nbRows || col >= b.nbCols {
		err = ErrTableOutOfBound
		return
	}
	return b.data[b.toIndex(row, col)], nil
}

func (b *TableBinding[T]) ItemAt(row, col int) (binding.Item[T], error) {
	if row >= b.nbRows || col >= b.nbCols {
		return nil, ErrTableOutOfBound
	}

	key := [2]int{row, col}

	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if we have a cached item for this position
	if item, exists := b.items[key]; exists {
		return item, nil
	}

	// Create new item and cache it
	item := &positionTrackingItem[T]{
		table: b,
		row:   row,
		col:   col,
	}

	b.items[key] = item

	// Add the item as a listener to the table so it gets notified when the table changes
	// Only add if not already in the list (shouldn't happen since we check cache first)
	b.listeners = append(b.listeners, item)

	return item, nil
}

// positionTrackingItem wraps a binding.Item to track its position in the table
// This allows the item to be looked up correctly even after the table is resized
type positionTrackingItem[T any] struct {
	table     *TableBinding[T]
	row, col  int
	listeners []binding.DataListener
}

func (pi *positionTrackingItem[T]) Get() (T, error) {
	// Always get from the current table position
	if pi.row >= pi.table.nbRows || pi.col >= pi.table.nbCols {
		var zero T
		return zero, ErrTableOutOfBound
	}
	return pi.table.data[pi.table.toIndex(pi.row, pi.col)], nil
}

func (pi *positionTrackingItem[T]) Set(val T) error {
	if pi.row >= pi.table.nbRows || pi.col >= pi.table.nbCols {
		return ErrTableOutOfBound
	}
	// Set the value in the table
	pi.table.lock.Lock()
	idx := pi.table.toIndex(pi.row, pi.col)
	pi.table.data[idx] = val
	pi.table.lock.Unlock()
	// Trigger table listeners (which will trigger this item's DataChanged, which will trigger item listeners)
	pi.table.trigger()
	return nil
}

func (pi *positionTrackingItem[T]) AddListener(listener binding.DataListener) {
	pi.listeners = append(pi.listeners, listener)
	listener.DataChanged()
}

func (pi *positionTrackingItem[T]) RemoveListener(listener binding.DataListener) {
	for i, l := range pi.listeners {
		if l == listener {
			pi.listeners = append(pi.listeners[:i], pi.listeners[i+1:]...)
			return
		}
	}
}

func (pi *positionTrackingItem[T]) trigger() {
	for _, l := range pi.listeners {
		l.DataChanged()
	}
}

// DataChanged is called when the table's data changes
// This is part of the binding.DataListener interface
// Note: This method may be called while the table's lock is already held,
// so we must not try to acquire the lock here.
func (pi *positionTrackingItem[T]) DataChanged() {
	// Don't trigger cell listeners from DataChanged
	// Cell listeners are triggered explicitly when needed (e.g., when position becomes invalid)
}

func (b *TableBinding[T]) toIndex(row, col int) int {
	return row*b.nbCols + col
}

func (b *TableBinding[T]) toIndexWithDims(row, col, cols int) int {
	return row*cols + col
}
