package widget

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type NumericalEntry[T uint64 | time.Duration] struct {
	widget.Entry
	baseUnit T
}

func NewNumericalEntry[T uint64 | time.Duration](baseUnit T) *NumericalEntry[T] {
	e := &NumericalEntry[T]{baseUnit: baseUnit}
	e.ExtendBaseWidget(e)
	return e
}

func (e *NumericalEntry[T]) TypedRune(r rune) {
	if r >= '0' && r <= '9' {
		e.Entry.TypedRune(r)
	}
}

func (e *NumericalEntry[T]) TypedShortcut(shortcut fyne.Shortcut) {
	paste, ok := shortcut.(*fyne.ShortcutPaste)
	if !ok {
		e.Entry.TypedShortcut(shortcut)
		return
	}

	content := paste.Clipboard.Content()
	if _, err := strconv.ParseInt(content, 10, 64); err == nil {
		e.Entry.TypedShortcut(shortcut)
	}
}

func (e *NumericalEntry[T]) Keyboard() mobile.KeyboardType {
	return mobile.NumberKeyboard
}

func (e *NumericalEntry[T]) Bind(data binding.Item[T]) {
	e.Entry.Bind(
		uiutils.NewBindMapper[T, string](data,
			func(sizeBytes T) string {
				inUint := sizeBytes / e.baseUnit
				return fmt.Sprintf("%d", inUint)
			},
			func(inNumLabel string) T {
				val, err := strconv.Atoi(inNumLabel)
				if err != nil {
					return 0
				}
				parsed := T(val)

				return parsed * e.baseUnit
			},
			func(sizeByte T, inNumLabel string) bool {
				val, err := strconv.Atoi(inNumLabel)
				if err != nil {
					return false
				}
				parsed := T(val)
				return parsed*e.baseUnit == sizeByte
			},
		),
	)
}
