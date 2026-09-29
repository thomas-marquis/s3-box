package uu

import (
	"errors"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/u"
)

var (
	ErrTableOutOfBound = errors.New("the specified coordinates are out of bound of the existing table")
)

type TableBinding[T any] struct {
	comparator      func(T, T) bool
	internalBinding binding.List[T]
	data            []T
	nbRows, nbCols  int
}

// NewTableBinding constructs a binding object that holds a 2-dimension data table.
func NewTableBinding[T any](comparator func(T, T) bool) *TableBinding[T] {
	b := &TableBinding[T]{
		comparator:      comparator,
		internalBinding: binding.NewList(comparator),
		data:            make([]T, 0),
	}

	b.internalBinding.AddListener(binding.NewDataListener(func() {
		// vals := u.SkipV(b.internalBinding.Get())
		// TODO: syn the data to the internal binding when its getting updated.
	}))

	return b
}

func (b *TableBinding[T]) AddListener(dl binding.DataListener) {
	b.internalBinding.AddListener(dl)
}

func (b *TableBinding[T]) RemoveListener(dl binding.DataListener) {
	b.internalBinding.RemoveListener(dl)
}

func (b *TableBinding[T]) Dims() (rows, cols int) {
	return b.nbRows, b.nbCols
}

// SetDim sets the table shape. It may alter the table content if the new size is larger than the previous one.
func (b *TableBinding[T]) SetDim(rows, cols int) {
	dataCpy := make([]T, len(b.data))
	copy(dataCpy, b.data)
}

func (b *TableBinding[T]) AppendRow(data []T) error {
	if b.nbCols == 0 {
		b.nbCols = len(data)
	}

	if len(data) != b.nbCols {
		return ErrTableOutOfBound
	}

	b.data = append(b.data, data...)
	b.nbRows++
	for j, d := range data {
		if err := b.internalBinding.Append(d); err != nil {
			return err
		}
		if di, err := b.internalBinding.GetItem(b.toIndex(b.nbRows-1, j)); err == nil {
			r, c := b.nbRows-1, j
			di.AddListener(binding.NewDataListener(func() {
				newVal, err := di.(binding.Item[T]).Get()
				if err != nil {
					return
				}
				// If the table dims has changed meanwhile
				if r >= b.nbRows || c >= b.nbCols {
					// this is not supposed to happened...
					// ... or that means the data item is dangling
					// (e.g. still in the memory but untied anymore to any of the table's items)
					// In such a case, we simply ignore it.
					return
				}
				idx := b.toIndex(r, c)
				b.data[idx] = newVal
			}))
		} else {
			return err
		}
	}

	return nil
}

func (b *TableBinding[T]) RemoveRow(row int) error {
	if row >= b.nbRows {
		return ErrTableOutOfBound
	}

	newData := make([]T, len(b.data)-b.nbCols)
	var iNext int
	for iPrev := range b.nbRows {
		if iPrev >= row && iPrev < b.nbRows-1 {
			for j := range b.nbCols {
				di, err := b.internalBinding.GetItem(b.toIndex(iPrev, j))
				if err != nil {
					return err
				}

				// shifting the values of the bound items located at the deleted row location or after
				shiftedVal := b.getValue(iPrev+1, j)
				if err := di.(binding.Item[T]).Set(shiftedVal); err != nil {
					return err
				}
			}
		}

		if iPrev == row {
			continue
		}

		for j := range b.nbCols {
			newData[b.toIndex(iNext, j)] = b.data[b.toIndex(iPrev, j)]
		}

		iNext++
	}

	for j := range b.nbCols {
		di, err := b.internalBinding.GetItem(b.toIndex(b.nbRows-1, j))
		if err != nil {
			return err
		}
		var zv T
		if err := di.(binding.Item[T]).Set(zv); err != nil {
			return err
		}
	}

	b.data = newData
	b.nbRows--

	if err := b.internalBinding.Set(b.data); err != nil {
		return err
	}

	return nil
}

func (b *TableBinding[T]) SetValue(val T, row, col int) error {
	if row >= b.nbRows || col >= b.nbCols {
		return ErrTableOutOfBound
	}

	b.setValue(val, row, col)

	return nil
}

func (b *TableBinding[T]) ValueAt(row, col int) (val T, err error) {
	if row >= b.nbRows || col >= b.nbCols {
		err = ErrTableOutOfBound
		return
	}
	return b.getValue(row, col), nil
}

func (b *TableBinding[T]) ItemAt(row, col int) (binding.Item[T], error) {
	if row >= b.nbRows || col >= b.nbCols {
		return nil, ErrTableOutOfBound
	}
	idx := b.toIndex(row, col)
	di, err := b.internalBinding.GetItem(idx)
	if err != nil {
		return nil, err
	}

	return di.(binding.Item[T]), nil
}

func (b *TableBinding[T]) setValue(val T, row, col int) {
	idx := b.toIndex(row, col)
	b.data[idx] = val
	u.Skip(b.internalBinding.SetValue(idx, val))
}

func (b *TableBinding[T]) getValue(row, col int) T {
	return b.data[b.toIndex(row, col)]
}

func (b *TableBinding[T]) toIndex(row, col int) int {
	return row*b.nbCols + col
}
