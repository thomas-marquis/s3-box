package components

import (
	appcontext "go2s3/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func NewConnectionDialog(
	ctx appcontext.AppContext,
	label, name, accessKey, secretKey, server, bucket string,
	useTLS, enableCopy bool,
	onSave func(name, accessKey, secretKey, server, bucket string, useTLS bool) error,
) *dialog.CustomDialog {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Connection name")
	nameEntry.SetText(name)

	accessKeyEntry := widget.NewEntry()
	accessKeyEntry.SetPlaceHolder("Access key")
	accessKeyEntry.SetText(accessKey)

	secretKeyEntry := widget.NewEntry()
	secretKeyEntry.SetPlaceHolder("secretKey")
	secretKeyEntry.SetText(secretKey)

	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("Server")
	serverEntry.SetText(server)

	bucketEntry := widget.NewEntry()
	bucketEntry.SetPlaceHolder("Bucket")
	bucketEntry.SetText(bucket)

	useTlsEntry := widget.NewCheck("Use TLS", nil)
	useTlsEntry.Checked = useTLS

	saveBtn := widget.NewButton("Save", func() {})
	saveBtn.SetIcon(theme.ConfirmIcon())

	copyAccessKeyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if enableCopy && accessKeyEntry.Text != "" {
			ctx.W().Clipboard().SetContent(accessKeyEntry.Text)
		}
	})

	copySecretKeyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if enableCopy && secretKeyEntry.Text != "" {
			ctx.W().Clipboard().SetContent(secretKeyEntry.Text)
		}
	})

	copyServerBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if enableCopy && serverEntry.Text != "" {
			ctx.W().Clipboard().SetContent(serverEntry.Text)
		}
	})

	copyBucketBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if enableCopy && bucketEntry.Text != "" {
			ctx.W().Clipboard().SetContent(bucketEntry.Text)
		}
	})

	if !enableCopy {
		copyAccessKeyBtn.Hide()
		copySecretKeyBtn.Hide()
		copyServerBtn.Hide()
		copyBucketBtn.Hide()
	}

	d := dialog.NewCustom(
		label,
		"Close",
		container.NewVBox(
			nameEntry,
			container.NewBorder(nil, nil, nil, copyAccessKeyBtn, accessKeyEntry),
			container.NewBorder(nil, nil, nil, copySecretKeyBtn, secretKeyEntry),
			container.NewBorder(nil, nil, nil, copyServerBtn, serverEntry),
			container.NewBorder(nil, nil, nil, copyBucketBtn, bucketEntry),
			useTlsEntry,
			container.NewHBox(saveBtn),
		),
		ctx.W(),
	)
	d.Resize(fyne.NewSize(500, 200))

	saveBtn.OnTapped = func() {
		err := onSave(
			nameEntry.Text,
			accessKeyEntry.Text,
			secretKeyEntry.Text,
			serverEntry.Text,
			bucketEntry.Text,
			useTlsEntry.Checked,
		)
		if err == nil {
			d.Hide()
			nameEntry.Text = ""
			accessKeyEntry.Text = ""
			secretKeyEntry.Text = ""
			serverEntry.Text = ""
			bucketEntry.Text = ""
			useTlsEntry.Checked = false
		}
	}

	return d
}
