package components_test

import (
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	"testing"

	"go.uber.org/mock/gomock"
)

func Benchmark_DirDetails_WithMutation(b *testing.B) {
	// Given
	ctrl := gomock.NewController(b)
	mCtx := mocks_appcontext.NewMockAppContext(ctrl)
	c := components.NewDirDetails()
	d := explorer.NewS3Directory("root", nil)
	b.ResetTimer()

	// When
	for i := 0; i < b.N; i++ {
		c.Update(mCtx, d)
	}
}
