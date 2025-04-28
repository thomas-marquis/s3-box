package components

import (
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func makeCopyBtn(enableCopy bool, entry *widget.Entry, w fyne.Window) *widget.Button {
	return widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if enableCopy && entry.Text != "" {
			w.Clipboard().SetContent(entry.Text)
		}
	})
}

func makeTextInput(label, defaultValue, placeholder string, enableCopy bool, w fyne.Window) (*fyne.Container, *widget.Entry) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(placeholder)
	entry.SetText(defaultValue)
	copyBtn := makeCopyBtn(enableCopy, entry, w)
	if !enableCopy {
		copyBtn.Hide()
	}
	entryLabel := widget.NewLabel(label)
	c := container.NewBorder(nil, nil, entryLabel, copyBtn, entry)
	return c, entry
}

func NewConnectionDialog(
	ctx appcontext.AppContext,
	label, name, accessKey, secretKey, server, bucket, region string,
	useTLS, enableCopy bool,
	onSave func(name, accessKey, secretKey, server, bucket, region string, useTLS bool) error,
) *dialog.CustomDialog {
	nameBloc, nameEntry := makeTextInput("Connection name", name, "My new connection", enableCopy, ctx.Window())
	accessKeyBloc, accessKeyEntry := makeTextInput("Access key Id", accessKey, "Access key", enableCopy, ctx.Window())
	secretKeyBloc, secretKeyEntry := makeTextInput("Secret access key", secretKey, "Secret key", enableCopy, ctx.Window())
	serverBloc, serverEntry := makeTextInput("Server hostname", server, "s3.amazonaws.com", enableCopy, ctx.Window())
	bucketBloc, bucketEntry := makeTextInput("Bucket name", bucket, "my-bucket", enableCopy, ctx.Window())
	regionBloc, regionEntry := makeTextInput("Region", region, "us-east-1", enableCopy, ctx.Window())

	useTlsEntry := widget.NewCheck("Use TLS", nil)
	useTlsEntry.Checked = useTLS

	connTypeChoice := widget.NewRadioGroup([]string{"AWS", "Custom"}, func(val string) {
		if val == "AWS" {
			useTlsEntry.Hide()
			serverEntry.SetText("")
			serverBloc.Hide()
			regionBloc.Show()
		} else {
			useTlsEntry.Show()
			serverBloc.Show()
			regionEntry.SetText("")
			regionBloc.Hide()
		}
	})
	if region == "" {
		connTypeChoice.SetSelected("Custom")
	} else {
		connTypeChoice.SetSelected("AWS")
	}

	saveBtn := widget.NewButton("Save", func() {})
	saveBtn.SetIcon(theme.ConfirmIcon())

	d := dialog.NewCustom(
		label,
		"Close",
		container.NewVBox(
			connTypeChoice,
			nameBloc,
			serverBloc,
			accessKeyBloc,
			secretKeyBloc,
			bucketBloc,
			regionBloc,
			useTlsEntry,
			container.NewHBox(saveBtn),
		),
		ctx.Window(),
	)
	d.Resize(fyne.NewSize(650, 200))

	saveBtn.OnTapped = func() {
		err := onSave(
			nameEntry.Text,
			accessKeyEntry.Text,
			secretKeyEntry.Text,
			serverEntry.Text,
			bucketEntry.Text,
			regionEntry.Text,
			useTlsEntry.Checked,
		)
		if err == nil {
			d.Hide()
			nameEntry.Text = ""
			accessKeyEntry.Text = ""
			secretKeyEntry.Text = ""
			serverEntry.Text = ""
			bucketEntry.Text = ""
			regionEntry.Text = ""
			useTlsEntry.Checked = false
		}
	}

	return d
}
