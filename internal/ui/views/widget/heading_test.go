package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

func TestHeading(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should display simple text", func(t *testing.T) {
		// When
		res := widget.NewHeading("Hello world!")
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		assert.Equal(t, "Hello world!", res.Text)
		fyne_test.AssertRendersToMarkup(t, "heading_static", c)
	})

	t.Run("should display dynamiq text", func(t *testing.T) {
		// Given
		data := binding.NewString()

		// When
		res := widget.NewHeadingWithData(data)
		data.Set("Hello world!") //nolint:errcheck
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "heading_dynamic", c)
	})
}
