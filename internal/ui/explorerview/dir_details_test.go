package explorerview

import (
	"testing"

	"github.com/thomas-marquis/s3-box/internal/explorer"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"

	"go.uber.org/mock/gomock"
)

func Benchmark_DirDetails_WithMutation(b *testing.B) {
	// Given
	ctrl := gomock.NewController(b)
	mCtx := mocks_appcontext.NewMockAppContext(ctrl)
	c := newDirDetails()
	d := explorer.NewDirectory("root", nil)
	b.ResetTimer()

	// When
	for i := 0; i < b.N; i++ {
		c.Update(mCtx, d)
	}
}
