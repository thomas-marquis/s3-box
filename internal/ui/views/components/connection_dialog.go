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

func makeCopyBtnWithData(enableCopy bool, data binding.String, w fyne.Window) *widget.Button {
	return widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if enableCopy {
			value, err := data.Get()
			if err == nil && value != "" {
				w.Clipboard().SetContent(value)
			}
		}
	})
}

func makeTextInputWithData(data binding.String, label, placeholder string, enableCopy bool, w fyne.Window) (*fyne.Container, *widget.Entry) {
	entry := widget.NewEntryWithData(data)
	entry.SetPlaceHolder(placeholder)
	copyBtn := makeCopyBtnWithData(enableCopy, data, w)
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
	onSave func(conn *connection.Connection) error,
) *dialog.CustomDialog {
	// Init data bindings
	nameData := binding.NewString()
	nameData.Set(defaultConn.Name)

	accessKeyData := binding.NewString()
	accessKeyData.Set(defaultConn.AccessKey)

	secretKeyData := binding.NewString()
	secretKeyData.Set(defaultConn.SecretKey)

	serverData := binding.NewString()
	serverData.Set(defaultConn.Server)

	bucketData := binding.NewString()
	bucketData.Set(defaultConn.BucketName)

	regionData := binding.NewString()
	regionData.Set(defaultConn.Region)

	readOnlyData := binding.NewBool()
	readOnlyData.Set(defaultConn.ReadOnly)

	useTlsData := binding.NewBool()
	useTlsData.Set(defaultConn.UseTls)

	connectionTypeData := binding.NewUntyped()
	connectionTypeData.Set(defaultConn.Type)

	nameBloc, nameEntry := makeTextInputWithData(
		nameData,
		"Connection name",
		"My new connection",
		enableCopy,
		ctx.Window(),
	)
	accessKeyBloc, accessKeyEntry := makeTextInputWithData(
		accessKeyData,
		"Access key Id",
		"Access key",
		enableCopy,
		ctx.Window(),
	)
	secretKeyBloc, secretKeyEntry := makeTextInputWithData(
		secretKeyData,
		"Secret access key",
		"Secret key",
		enableCopy,
		ctx.Window(),
	)
	serverBloc, serverEntry := makeTextInputWithData(
		serverData,
		"Server hostname",
		"s3.amazonaws.com",
		enableCopy,
		ctx.Window(),
	)
	bucketBloc, bucketEntry := makeTextInputWithData(
		bucketData,
		"Bucket name",
		"my-bucket",
		enableCopy,
		ctx.Window(),
	)
	regionBloc, regionEntry := makeTextInputWithData(
		regionData,
		"Region",
		"us-east-1",
		enableCopy,
		ctx.Window(),
	)
	useTlsCheckbox := widget.NewCheckWithData("Use TLS", useTlsData)
	readOnlyCheckbox := widget.NewCheckWithData("Read only", readOnlyData)

	connTypeRadio := widget.NewRadioGroup([]string{"AWS", "Other"}, func(val string) {
		var selectedConnectionType connection.ConnectionType
		switch val {
		case "AWS":
			selectedConnectionType = connection.AWSConnectionType
			useTlsCheckbox.Hide()
			serverData.Set("")
			serverBloc.Hide()
			regionBloc.Show()
		case "Other":
			selectedConnectionType = connection.S3LikeConnectionType
			useTlsCheckbox.Show()
			serverBloc.Show()
			regionData.Set("")
			regionBloc.Hide()
		default:
			panic("Unknown connection type")
		}
		connectionTypeData.Set(selectedConnectionType)
	})

	switch defaultConn.Type {
	case connection.AWSConnectionType:
		connTypeRadio.SetSelected("AWS")
	case connection.S3LikeConnectionType:
		connTypeRadio.SetSelected("Other")
	}

	// init form
	saveBtn := widget.NewButton("Save", func() {})
	saveBtn.SetIcon(theme.ConfirmIcon())

	d := dialog.NewCustom(
		label,
		"Close",
		container.NewVBox(
			connTypeRadio,
			nameBloc,
			serverBloc,
			accessKeyBloc,
			secretKeyBloc,
			bucketBloc,
			regionBloc,
			useTlsCheckbox,
			readOnlyCheckbox,
			container.NewHBox(saveBtn),
		),
		ctx.Window(),
	)
	d.Resize(fyne.NewSize(650, 200))

	saveBtn.OnTapped = func() {
		di, _ := connectionTypeData.Get()
		selectedConnType, ok := di.(connection.ConnectionType)
		if !ok {
			panic("Invalid connection type")
		}
		isReadOnlySelected, _ := readOnlyData.Get()

		var newConn *connection.Connection
		switch selectedConnType {
		case connection.AWSConnectionType:
			newConn = connection.NewConnection(
				nameEntry.Text,
				accessKeyEntry.Text,
				secretKeyEntry.Text,
				bucketEntry.Text,
				connection.AsAWSConnection(regionEntry.Text),
				connection.WithReadOnlyOption(isReadOnlySelected),
			)
		case connection.S3LikeConnectionType:
			newConn = connection.NewConnection(
				nameEntry.Text,
				accessKeyEntry.Text,
				secretKeyEntry.Text,
				bucketEntry.Text,
				connection.AsS3LikeConnection(serverEntry.Text, useTlsCheckbox.Checked),
				connection.WithReadOnlyOption(isReadOnlySelected),
			)
		default:
			panic("Unknown connection type")
		}

		if err := onSave(newConn); err == nil {
			d.Hide()
			nameData.Set("")
			accessKeyData.Set("")
			secretKeyData.Set("")
			serverData.Set("")
			bucketData.Set("")
			regionData.Set("")
			useTlsData.Set(false)
			connectionTypeData.Set(connection.AWSConnectionType)
			readOnlyData.Set(false)
		}
	}

	return d
}
