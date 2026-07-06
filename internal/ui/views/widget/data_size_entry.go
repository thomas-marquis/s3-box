package widget

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type DataSizeEntry struct {
	widget.Entry
	baseUnit uint64
}

func NewDataSizeEntry(baseUnit uint64) *DataSizeEntry {
	e := &DataSizeEntry{baseUnit: baseUnit}
	e.ExtendBaseWidget(e)
	return e
}

func (e *DataSizeEntry) TypedRune(r rune) {
	if r >= '0' && r <= '9' {
		e.Entry.TypedRune(r)
	}
}

func (e *DataSizeEntry) TypedShortcut(shortcut fyne.Shortcut) {
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

func (e *DataSizeEntry) Keyboard() mobile.KeyboardType {
	return mobile.NumberKeyboard
}

func (e *DataSizeEntry) Bind(data binding.Item[uint64]) {
	e.Entry.Bind(
		uiutils.NewBindMapper[uint64, string](data,
			func(sizeBytes uint64) string {
				inUint := sizeBytes / e.baseUnit
				return fmt.Sprintf("%d", inUint)
			},
			func(inUnitLabel string) uint64 {
				inUnit, err := strconv.ParseUint(inUnitLabel, 10, 64)
				if err != nil {
					return 0
				}
				return inUnit * e.baseUnit
			},
			func(sizeByte uint64, inUnitLabel string) bool {
				inUnit, err := strconv.ParseUint(inUnitLabel, 10, 64)
				if err != nil {
					return false
				}
				return inUnit*e.baseUnit == sizeByte
			},
		),
	)
}
