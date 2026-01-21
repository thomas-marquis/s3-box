package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	fyne_widget "fyne.io/fyne/v2/widget"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

type tappableLabel struct {
	*fyne_widget.Label

	window fyne.Window
}

var (
	_ fyne.Widget         = (*tappableLabel)(nil)
	_ fyne.DoubleTappable = (*tappableLabel)(nil)
	_ fyne.Tappable       = (*tappableLabel)(nil)
	_ desktop.Cursorable  = (*tappableLabel)(nil)
)

func newTappableLabel(text string, window fyne.Window) *tappableLabel {
	l := &tappableLabel{fyne_widget.NewLabel(text), window}
	l.ExtendBaseWidget(l)
	return l
}

func (l *tappableLabel) DoubleTapped(*fyne.PointEvent) {
	l.handleTape()
}

func (l *tappableLabel) Tapped(*fyne.PointEvent) {
	l.handleTape()
}

func (l *tappableLabel) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (l *tappableLabel) handleTape() {
	content := fyne_widget.NewLabel(l.Text)
	content.Wrapping = fyne.TextWrapWord
	content.Alignment = fyne.TextAlignLeading
	content.Selectable = true

	d := dialog.NewCustom("", "Ok", content, l.window)
	d.Resize(fyne.NewSize(600, 300))
	d.Show()
}

func GetNotificationView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	notifications := appCtx.NotificationViewModel().Notifications()
	notificationList := fyne_widget.NewListWithData(
		notifications,
		func() fyne.CanvasObject {
			label := newTappableLabel("", appCtx.Window())
			label.Alignment = fyne.TextAlignLeading
			label.Truncation = fyne.TextTruncateEllipsis
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Selectable = false
			return label
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			value, _ := di.(binding.String)
			label := obj.(*tappableLabel)
			label.Bind(value)
		},
	)

	nothingToDisplay := container.NewCenter(
		fyne_widget.NewLabelWithStyle("Noting to display at the moment...",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true}))
	notificationList.Hide()

	notifications.AddListener(binding.NewDataListener(func() {
		if notifications.Length() == 0 {
			notificationList.Hide()
			nothingToDisplay.Show()
		} else {
			notificationList.Show()
			nothingToDisplay.Hide()
		}
	}))

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeading("Notifications"),
			fyne_widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(
			notificationList,
		),
		nothingToDisplay,
	), nil
}
