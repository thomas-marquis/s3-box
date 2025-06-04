package uiutils

import (
	"fmt"

	"fyne.io/fyne/v2/data/binding"
)

// GetString retrieves a string value from a binding.String.
// This fnction panics if there is an error while getting the value.
func GetString(data binding.String) string {
	value, err := data.Get()
	if err != nil {
		panic("error while getting string from binding")
	}
	return value
}

// GetInt retrieves a boolean value from a binding.Bool.
// This function panics if there is an error while getting the value.
func GetBool(data binding.Bool) bool {
	value, err := data.Get()
	if err != nil {
		panic("error while getting string from binding")
	}
	return value
}

// GetInt retrieves and cast any value from a binding.Untyped acording to the generic type specified.
// This function panics if there is an error while getting the value.
// If the value is not of the expected type, it will panic with an error message.
func GetUntypedOrPanic[T any](data binding.Untyped) T {
	di, err := data.Get()
	if err != nil {
		panic("error while getting value from binding")
	}
	value, ok := di.(T)
	if !ok {
		panic("Invalid casting type for binding.Untyped")
	}
	return value
}

func GetUntypedListOrPanic[T any](data binding.UntypedList) []T {
	dis, err := data.Get()
	if err != nil {
		panic("error while getting value from binding")
	}

	values := make([]T, len(dis))
	for i, di := range dis {
		value, ok := di.(T)
		if !ok {
			panic("Invalid casting type for binding.UntypedList")
		}
		values[i] = value
	}

	return values
}

func GetUntypedList[T any](data binding.UntypedList) ([]T, error) {
	items, err := data.Get()
	if err != nil {
		return nil, err
	}

	values := make([]T, len(items))
	for i, item := range items {
		value, ok := item.(T)
		if !ok {
			return nil, fmt.Errorf("invalid casting type for binding.UntypedList: expected %T, got %T", value, item)
		}
		values[i] = value
	}

	return values, nil
}
