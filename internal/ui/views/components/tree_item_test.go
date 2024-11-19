package components_test

import (
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TreeItem_ShouldUpdateWithoutPanic(t *testing.T) {
	// Given
	builder := components.NewTreeItemBuilder()
	container := builder.NewRaw()

	// When / Then
	assert.NotPanics(t, func() {
		builder.Update(container, "test")
	})
}
