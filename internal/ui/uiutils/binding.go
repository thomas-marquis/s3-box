package uiutils

import (
	"fmt"
	"runtime"
	"time"

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

type baseSettingsBinding[T any] struct {
	binding.Item[T]
	cancel func()
}

func (b *baseSettingsBinding[T]) bind(s *settings.Settings, name string) {
	b.cancel = s.Observe(name, func(value any) {
		if val, ok := value.(T); ok {
			b.Set(val) //nolint:errcheck
		}
	})
	runtime.AddCleanup(b, func(cancel func()) {
		cancel()
	}, b.cancel)

	b.AddListener(binding.NewDataListener(func() {
		val, err := b.Get()
		if err != nil {
			return
		}
		if writeErr := s.Write(name, val); writeErr != nil {
			return
		}
	}))
}

// SettingsBindingString provides two-way synchronization between a string binding and a string setting.
type SettingsBindingString struct {
	binding.String
	cancel func()
}

// NewSettingsBindingString creates a two-way binding between a string binding and a string setting.
// Changes to the setting (via WriteSucceeded events) will update the binding.
// Changes to the binding will write to the setting.
func NewSettingsBindingString(s *settings.Settings, name string) binding.String {
	bs := &SettingsBindingString{}
	bs.String = binding.NewString()

	if !s.IsExistsWithType(name, settings.StringType) {
		return bs
	}

	initialValue := s.ReadString(name)
	bs.Set(initialValue) //nolint:errcheck

	bs.cancel = s.Observe(name, func(value any) {
		if strVal, ok := value.(string); ok {
			bs.Set(strVal) //nolint:errcheck
		}
	})
	runtime.AddCleanup(bs, func(cancel func()) {
		cancel()
	}, bs.cancel)

	bs.AddListener(binding.NewDataListener(func() {
		val, err := bs.Get()
		if err != nil {
			return
		}
		if writeErr := s.Write(name, val); writeErr != nil {
			return
		}
	}))

	return bs
}

// SettingsBindingDuration provides two-way synchronization between a float binding and a duration setting.
type SettingsBindingDuration struct {
	binding.Float
	cancel func()
}

// NewSettingsBindingDuration creates a two-way binding between a float64 binding (representing seconds)
// and a duration setting. This is useful for UI elements that work with numeric time values.
func NewSettingsBindingDuration(s *settings.Settings, name string) binding.Float {
	bf := &SettingsBindingDuration{}
	bf.Float = binding.NewFloat()

	if !s.IsExistsWithType(name, settings.DurationType) {
		return bf
	}

	initialValue := s.ReadDuration(name)
	bf.Set(initialValue.Seconds()) //nolint:errcheck

	bf.cancel = s.Observe(name, func(value any) {
		durationVal, ok := value.(time.Duration)
		if ok {
			bf.Set(durationVal.Seconds()) //nolint:errcheck
		}
	})
	runtime.AddCleanup(bf, func(cancel func()) {
		cancel()
	}, bf.cancel)

	bf.AddListener(binding.NewDataListener(func() {
		val, err := bf.Get()
		if err != nil {
			return
		}
		if writeErr := s.Write(name, time.Duration(val)*time.Second); writeErr != nil {
			// Setting not ready
			return
		}
	}))

	return bf
}

// SettingsBindingIntForDuration provides two-way synchronization between an int binding and a duration setting.
// The binding stores int values (representing whole seconds), but they are converted to/from time.Duration for the setting.
type SettingsBindingIntForDuration struct {
	binding.Int
	cancel func()
}

// NewSettingsBindingIntForDuration creates a two-way binding between an int binding and a duration setting.
// The binding stores int values (whole seconds), which are converted to/from time.Duration for the setting.
func NewSettingsBindingIntForDuration(s *settings.Settings, name string) binding.Int {
	bi := &SettingsBindingIntForDuration{}
	bi.Int = binding.NewInt()

	if !s.IsExistsWithType(name, settings.DurationType) {
		return bi
	}

	initialValue := s.ReadDuration(name)
	bi.Set(int(initialValue.Seconds())) //nolint:errcheck

	bi.cancel = s.Observe(name, func(value any) {
		durationVal, ok := value.(time.Duration)
		if ok {
			bi.Set(int(durationVal.Seconds())) //nolint:errcheck
		}
	})
	runtime.AddCleanup(bi, func(cancel func()) {
		cancel()
	}, bi.cancel)

	bi.AddListener(binding.NewDataListener(func() {
		val, err := bi.Get()
		if err != nil {
			return
		}
		if writeErr := s.Write(name, time.Duration(val)*time.Second); writeErr != nil {
			// Setting not ready - could retry or notify user
			return
		}
	}))

	return bi
}

// SettingsBindingIntToUint64 provides two-way synchronization between an int binding (in KB) and a uint64 setting (in bytes).
// The binding stores int values (representing KB), but they are converted to/from uint64 bytes for the setting.
type SettingsBindingIntToUint64 struct {
	baseSettingsBinding[uint64]
}

// NewSettingsBindingIntToUint64 creates a two-way binding between an int binding (in KB) and a uint64 setting (in bytes).
// The binding stores int values, which are converted to/from uint64 bytes for the setting.
func NewSettingsBindingIntToUint64(s *settings.Settings, name string) binding.Item[uint64] {
	bi := &SettingsBindingIntToUint64{
		baseSettingsBinding[uint64]{
			Item: binding.NewItem[uint64](func(u1, u2 uint64) bool {
				return u1 == u2
			}),
		},
	}

	if !s.IsExistsWithType(name, settings.Uint64Type) {
		return bi
	}

	initialValue := s.ReadUint64(name)
	bi.Set(initialValue) //nolint:errcheck

	bi.bind(s, name)

	return bi
}

type BindMapper[S, T any] struct {
	binding.Item[T]
	src binding.Item[S]
}

func NewBindMapper[S, T any](src binding.Item[S],
	sToT func(S) T,
	tToS func(T) S,
	comparator func(S, T) bool,
) *BindMapper[S, T] {
	b := &BindMapper[S, T]{
		Item: binding.NewItem(func(x, y T) bool {
			return comparator(tToS(x), y)
		}),
		src: src,
	}

	srcDl := binding.NewDataListener(func() {
		newVal, err := src.Get()
		if err != nil {
			return
		}
		curr, err := b.Get()
		if err != nil || comparator(newVal, curr) {
			return
		}
		b.Set(sToT(newVal)) //nolint:errcheck
	})
	src.AddListener(srcDl)

	runtime.AddCleanup(b, func(s binding.Item[S]) {
		s.RemoveListener(srcDl)
	}, src)

	b.AddListener(binding.NewDataListener(func() {
		newVal, err := b.Get()
		if err != nil {
			return
		}
		curr, err := src.Get()
		if err != nil || comparator(curr, newVal) {
			return
		}
		src.Set(tToS(newVal)) //nolint:errcheck
	}))

	return b
}
