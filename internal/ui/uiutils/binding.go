package uiutils

import (
	"fmt"
	"runtime"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

// GetString retrieves a string value from a binding.String.
// This function panics if there is an error while getting the value.
func GetString(data binding.String) string {
	value, err := data.Get()
	if err != nil {
		panic("error while getting string from binding")
	}
	return value
}

// GetBool retrieves a boolean value from a binding.Bool.
// This function panics if there is an error while getting the value.
func GetBool(data binding.Bool) bool {
	value, err := data.Get()
	if err != nil {
		panic("error while getting string from binding")
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

type BindingItemFormatter[T any] struct {
	binding.String

	original   binding.Item[T]
	formatFunc func(T) string
}

func NewBindingItemFormatter[T any](original binding.Item[T], formatFunc func(T) string) binding.String {
	b := &BindingItemFormatter[T]{
		String:     binding.NewString(),
		original:   original,
		formatFunc: formatFunc,
	}

	original.AddListener(binding.NewDataListener(func() { // TODO: possible leak...
		item, _ := b.original.Get()
		b.Set(b.formatFunc(item)) //nolint:errcheck
	}))
	return b
}

type SettingsStringBinding struct {
	binding.String

	aggregate *settings.SettingsV3
}

func NewSettingsBindingString(s *settings.SettingsV3, name string) binding.String {
	bs := binding.NewString()
	b := &SettingsStringBinding{
		String:    bs,
		aggregate: s,
	}

	if !s.IsExistsWithType(name, settings.StringType) {
		return b
	}

	cancel := s.Observe(name, func(value any) {
		bs.Set(value.(string)) //nolint:errcheck
	})
	runtime.AddCleanup(b, func(cancel func()) {
		cancel()
	}, cancel)

	return b
}
