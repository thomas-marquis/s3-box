package widget_test

import (
	"testing"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	"go.uber.org/mock/gomock"
)

func TestConnectionForm(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fyne_test.NewApp()
	mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
	mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()

	deck := connection_deck.New()
	conn := deck.New("Test", "ak", "sk", "bucket").Connection()

	t.Run("should display AWS form by default", func(t *testing.T) {
		// When
		res := widget.NewConnectionForm(mockAppCtx, conn, true,
			func(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) {})
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "connection_form_aws", c)
	})

	t.Run("should display S3-like form", func(t *testing.T) {
		// Given
		conn.AsS3Like("http://localhost:9000", false)

		// When
		res := widget.NewConnectionForm(mockAppCtx, conn, true,
			func(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) {})
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "connection_form_s3_like", c)
	})
}
