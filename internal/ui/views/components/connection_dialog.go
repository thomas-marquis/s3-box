package components

import (
	"github.com/thomas-marquis/s3-box/internal/connection"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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
	label string,
	defaultConn connection.Connection,
	enableCopy bool,
	onSave func(name, accessKey, secretKey, server, bucket, region string, useTLS, readOnly bool, connectionType connection.ConnectionType) error,
) *dialog.CustomDialog {
	nameBloc, nameEntry := makeTextInput(
		"Connection name",
		defaultConn.Name,
		"My new connection",
		enableCopy,
		ctx.Window(),
	)
	accessKeyBloc, accessKeyEntry := makeTextInput(
		"Access key Id",
		defaultConn.AccessKey,
		"Access key",
		enableCopy,
		ctx.Window(),
	)
	secretKeyBloc, secretKeyEntry := makeTextInput(
		"Secret access key",
		defaultConn.SecretKey,
		"Secret key",
		enableCopy,
		ctx.Window(),
	)
	serverBloc, serverEntry := makeTextInput(
		"Server hostname",
		defaultConn.Server,
		"s3.amazonaws.com",
		enableCopy,
		ctx.Window(),
	)
	bucketBloc, bucketEntry := makeTextInput(
		"Bucket name",
		defaultConn.BucketName,
		"my-bucket",
		enableCopy,
		ctx.Window(),
	)
	regionBloc, regionEntry := makeTextInput(
		"Region",
		defaultConn.Region,
		"us-east-1",
		enableCopy,
		ctx.Window(),
	)

	useTlsEntry := widget.NewCheck("Use TLS", nil)
	useTlsEntry.Checked = defaultConn.UseTls

	readOnlyData := binding.NewBool()
	readOnlyData.Set(defaultConn.ReadOnly)
	readOnlyCheckbox := widget.NewCheckWithData("Read only", readOnlyData)

	connectionType := binding.NewString()
	connTypeChoice := widget.NewRadioGroup([]string{"AWS", "Other"}, func(val string) {
		connectionType.Set(val)
		switch val {
		case "AWS":
			useTlsEntry.Hide()
			serverEntry.SetText("")
			serverBloc.Hide()
			regionBloc.Show()
		case "Other":
			useTlsEntry.Show()
			serverBloc.Show()
			regionEntry.SetText("")
			regionBloc.Hide()
		}
	})
	switch defaultConn.Type {
	case connection.AWSConnectionType:
		connTypeChoice.SetSelected("AWS")
	case connection.S3LikeConnectionType:
		connTypeChoice.SetSelected("Other")
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
			readOnlyCheckbox,
			container.NewHBox(saveBtn),
		),
		ctx.Window(),
	)
	d.Resize(fyne.NewSize(650, 200))

	saveBtn.OnTapped = func() {
		selectedConnType, _ := connectionType.Get()
		isReadOnlySelected, _ := readOnlyData.Get()
		err := onSave(
			nameEntry.Text,
			accessKeyEntry.Text,
			secretKeyEntry.Text,
			serverEntry.Text,
			bucketEntry.Text,
			regionEntry.Text,
			useTlsEntry.Checked,
			isReadOnlySelected,
			connection.NewConnectionTypeFromString(selectedConnType),
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
			connectionType.Set("AWS")
			readOnlyData.Set(false)
		}
	}

	return d
}
