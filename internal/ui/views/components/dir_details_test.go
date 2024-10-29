package components_test

import (
	"go2s3/internal/explorer"
	"go2s3/internal/ui/views/components"
	mocks_appcontext "go2s3/mocks/context"
	"testing"

	"go.uber.org/mock/gomock"
)

func Benchmark_DirDetails_WithMutation(b *testing.B) {
	// Given
	ctrl := gomock.NewController(b)
	mCtx := mocks_appcontext.NewMockAppContext(ctrl)
	c := components.NewDirDetails()
	d := explorer.NewDirectory("root", nil)
	b.ResetTimer()

	// When
	for i := 0; i < b.N; i++ {
		c.Update(mCtx, d)
	}
}
