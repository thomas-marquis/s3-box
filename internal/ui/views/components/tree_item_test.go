package components_test

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"

	"github.com/stretchr/testify/assert"
)

func Test_TreeItem_ShouldUpdateWithoutPanic(t *testing.T) {
	// Given
	builder := components.NewTreeItemBuilder()
	container := builder.NewRaw()
	nodeItem := viewmodel.NewTreeNode("/home/", "home", true)

	// When / Then
	assert.NotPanics(t, func() {
		builder.Update(container, *nodeItem)
	})
}

func Test_TreeItem_ShouldUpdateWithIcon(t *testing.T) {
	// Given
	builder := components.NewTreeItemBuilder()
	container := builder.NewRaw()
	nodeItem := viewmodel.NewTreeNode("/home/", "home", true)

	// When
	builder.Update(container, *nodeItem)

	// Then
	assert.True(t, container.Objects[0].(*widget.Icon).Visible())
}