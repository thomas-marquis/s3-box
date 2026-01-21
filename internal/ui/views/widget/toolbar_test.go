package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

func TestToolbarButton(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should display button with text and icon", func(t *testing.T) {
		// Given
		tapped := false
		onTapped := func() { tapped = true }
		res := widget.NewToolbarButton("Test", theme.ConfirmIcon(), onTapped)
		c := fyne_test.NewWindow(res.ToolbarObject()).Canvas()

		// Then
		assert.Equal(t, "Test", res.Text)
		assert.Equal(t, theme.ConfirmIcon(), res.Icon)

		// When
		fyne_test.Tap(res.ToolbarObject().(fyne.Tappable))

		// Then
		assert.True(t, tapped)
		fyne_test.AssertRendersToMarkup(t, "toolbar_button", c)
	})

	t.Run("should be able to enable and disable", func(t *testing.T) {
		// Given
		res := widget.NewToolbarButton("Test", theme.ConfirmIcon(), func() {})

		// When
		res.Disable()

		// Then
		assert.True(t, res.ToolbarObject().(interface{ Disabled() bool }).Disabled())

		// When
		res.Enable()

		// Then
		assert.False(t, res.ToolbarObject().(interface{ Disabled() bool }).Disabled())
	})
}
