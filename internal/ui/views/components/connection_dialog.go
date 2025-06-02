package components

import (
	"github.com/thomas-marquis/s3-box/internal/connection"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"

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

func makeTextFormItemWithData(data binding.String, label, placeholder string, enableCopy bool, w fyne.Window) *widget.FormItem {
	entry := widget.NewEntryWithData(data)
	entry.SetPlaceHolder(placeholder)
	formItem := widget.NewFormItem(label, entry)
	if enableCopy {
		entry.ActionItem = makeCopyBtnWithData(enableCopy, data, w)
	}

	return formItem
}

func buildAWSForm(
	ctx appcontext.AppContext,
	defaultConn connection.Connection,
	enableCopy bool,
	onSubmit func(conn *connection.Connection) error,
) *widget.Form {
	// Init data bindings
	nameData := binding.NewString()
	nameData.Set(defaultConn.Name)

	accessKeyData := binding.NewString()
	accessKeyData.Set(defaultConn.AccessKey)

	secretKeyData := binding.NewString()
	secretKeyData.Set(defaultConn.SecretKey)

	bucketData := binding.NewString()
	bucketData.Set(defaultConn.BucketName)

	regionData := binding.NewString()
	regionData.Set(defaultConn.Region)

	readOnlyData := binding.NewBool()
	readOnlyData.Set(defaultConn.ReadOnly)

	// Create Form items
	nameFormItem := makeTextFormItemWithData(
		nameData,
		"Connection name",
		"My new connection",
		enableCopy,
		ctx.Window(),
	)
	accessKeyFormItem := makeTextFormItemWithData(
		accessKeyData,
		"Access key Id",
		"Access key",
		enableCopy,
		ctx.Window(),
	)
	secretKeyFormItem := makeTextFormItemWithData(
		secretKeyData,
		"Secret access key",
		"Secret key",
		enableCopy,
		ctx.Window(),
	)

	bucketFormItem := makeTextFormItemWithData(
		bucketData,
		"Bucket name",
		"my-bucket",
		enableCopy,
		ctx.Window(),
	)
	regionFormItem := makeTextFormItemWithData(
		regionData,
		"Region",
		"us-east-1",
		enableCopy,
		ctx.Window(),
	)

	readOnlyCheckbox := widget.NewCheckWithData("Read only", readOnlyData)
	readOnlyFormItem := widget.NewFormItem("Read only", readOnlyCheckbox)

	f := widget.NewForm(
		nameFormItem,
		accessKeyFormItem,
		secretKeyFormItem,
		bucketFormItem,
		regionFormItem,
		readOnlyFormItem,
	)
	f.OnSubmit = func() {
		newConn := connection.NewConnection(
			uiutils.GetString(nameData),
			uiutils.GetString(accessKeyData),
			uiutils.GetString(secretKeyData),
			uiutils.GetString(bucketData),
			connection.AsAWSConnection(uiutils.GetString(regionData)),
			connection.WithReadOnlyOption(uiutils.GetBool(readOnlyData)),
		)

		if err := onSubmit(newConn); err == nil {
			nameData.Set("")
			accessKeyData.Set("")
			secretKeyData.Set("")
			bucketData.Set("")
			regionData.Set("")
			readOnlyData.Set(false)
		}
	}

	return f
}

func buildS3LikeForm(
	ctx appcontext.AppContext,
	defaultConn connection.Connection,
	enableCopy bool,
	onSubmit func(conn *connection.Connection) error,
) *widget.Form {
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

	readOnlyData := binding.NewBool()
	readOnlyData.Set(defaultConn.ReadOnly)

	useTlsData := binding.NewBool()
	useTlsData.Set(defaultConn.UseTls)

	nameFormItem := makeTextFormItemWithData(
		nameData,
		"Connection name",
		"My new connection",
		enableCopy,
		ctx.Window(),
	)
	accessKeyFormItem := makeTextFormItemWithData(
		accessKeyData,
		"Access key Id",
		"Access key",
		enableCopy,
		ctx.Window(),
	)
	secretKeyFormItem := makeTextFormItemWithData(
		secretKeyData,
		"Secret access key",
		"Secret key",
		enableCopy,
		ctx.Window(),
	)
	serverFormItem := makeTextFormItemWithData(
		serverData,
		"Server hostname",
		"s3.amazonaws.com",
		enableCopy,
		ctx.Window(),
	)
	bucketFormItem := makeTextFormItemWithData(
		bucketData,
		"Bucket name",
		"my-bucket",
		enableCopy,
		ctx.Window(),
	)

	useTlsCheckbox := widget.NewCheckWithData("Use TLS", useTlsData)
	useTlsFormItem := widget.NewFormItem("UseTls", useTlsCheckbox)

	readOnlyCheckbox := widget.NewCheckWithData("Read only", readOnlyData)
	readOnlyFormItem := widget.NewFormItem("Read only", readOnlyCheckbox)

	// Create form
	f := widget.NewForm(
		nameFormItem,
		accessKeyFormItem,
		secretKeyFormItem,
		serverFormItem,
		bucketFormItem,
		useTlsFormItem,
		readOnlyFormItem,
	)
	f.OnSubmit = func() {
		newConn := connection.NewConnection(
			uiutils.GetString(nameData),
			uiutils.GetString(accessKeyData),
			uiutils.GetString(secretKeyData),
			uiutils.GetString(bucketData),
			connection.AsS3LikeConnection(uiutils.GetString(serverData), uiutils.GetBool(useTlsData)),
			connection.WithReadOnlyOption(uiutils.GetBool(readOnlyData)),
		)

		if err := onSubmit(newConn); err == nil {
			nameData.Set("")
			accessKeyData.Set("")
			secretKeyData.Set("")
			serverData.Set("")
			bucketData.Set("")
			useTlsData.Set(false)
			readOnlyData.Set(false)
		}
	}

	return f
}

func NewConnectionDialog(
	ctx appcontext.AppContext,
	label string,
	defaultConn connection.Connection,
	enableCopy bool,
	onSave func(conn connection.Connection) error,
) dialog.Dialog {
	var d dialog.Dialog

	handleOnSubmit := func(c *connection.Connection) error {
		err := onSave(*c)
		d.Hide()
		return err
	}

	awsForm := buildAWSForm(ctx, defaultConn, enableCopy, handleOnSubmit)
	awsForm.SubmitText = "Save"

	s3LikeForm := buildS3LikeForm(ctx, defaultConn, enableCopy, handleOnSubmit)
	s3LikeForm.SubmitText = "Save"

	tabs := container.NewAppTabs(
		container.NewTabItem(
			"AWS",
			container.NewVBox(
				widget.NewLabel(""),
				awsForm,
			),
		),
		container.NewTabItem(
			"Other (S3 Like)",
			container.NewVBox(
				widget.NewLabel(""),
				s3LikeForm,
			),
		),
	)

	switch defaultConn.Type {
	case connection.AWSConnectionType:
		tabs.SelectIndex(0)
	case connection.S3LikeConnectionType:
		tabs.SelectIndex(1)
	}

	d = dialog.NewCustom(label, "cancel", tabs, ctx.Window())
	d.Resize(fyne.NewSize(650, 200))

	return d
}
