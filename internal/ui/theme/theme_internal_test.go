package theme

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

func TestGetByName(t *testing.T) {
	t.Run("should returns dark theme", func(t *testing.T) {
		th := GetByName(settings.ColorThemeDark)
		assert.IsType(t, &appThemeDark{}, th)
	})

	t.Run("should returns light theme", func(t *testing.T) {
		th := GetByName(settings.ColorThemeLight)
		assert.IsType(t, &appThemeLight{}, th)
	})

	t.Run("should returns default system theme", func(t *testing.T) {
		th := GetByName(settings.ColorThemeSystem)
		assert.Equal(t, theme.DefaultTheme(), th)
	})
}

func TestGetDefaultByVariant(t *testing.T) {
	t.Run("should returns dark theme", func(t *testing.T) {
		th := GetDefaultByVariant(theme.VariantDark)
		assert.IsType(t, &appThemeDark{}, th)
	})

	t.Run("should returns light theme", func(t *testing.T) {
		th := GetDefaultByVariant(theme.VariantLight)
		assert.IsType(t, &appThemeLight{}, th)
	})
}

func TestHexToNRGBA(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected color.NRGBA
	}{
		{
			name:     "valid 6-digit hex with hash",
			hex:      "#FF5733",
			expected: color.NRGBA{R: 255, G: 87, B: 51, A: 255},
		},
		{
			name:     "valid 6-digit hex without hash",
			hex:      "00AABB",
			expected: color.NRGBA{R: 0, G: 170, B: 187, A: 255},
		},
		{
			name:     "valid 8-digit hex with hash",
			hex:      "#FF573380",
			expected: color.NRGBA{R: 255, G: 87, B: 51, A: 128},
		},
		{
			name:     "valid 8-digit hex without hash",
			hex:      "00AABBFF",
			expected: color.NRGBA{R: 0, G: 170, B: 187, A: 255},
		},
		{
			name:     "invalid length (short)",
			hex:      "#123",
			expected: color.NRGBA{},
		},
		{
			name:     "invalid length (long)",
			hex:      "#123456789",
			expected: color.NRGBA{},
		},
		{
			name:     "invalid characters (6-digit)",
			hex:      "#GGGGGG",
			expected: color.NRGBA{},
		},
		{
			name:     "invalid characters (8-digit)",
			hex:      "#FFFFFFFFGG",
			expected: color.NRGBA{},
		},
		{
			name:     "empty string",
			hex:      "",
			expected: color.NRGBA{},
		},
		{
			name:     "only hash",
			hex:      "#",
			expected: color.NRGBA{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hexToNRGBA(tt.hex)
			assert.Equal(t, tt.expected, result)
		})
	}
}
