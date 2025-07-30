package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type ConnDialogOnSubmitFunc func(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) error

type ConnectionForm struct {
	widget.BaseWidget
	appCtx            appcontext.AppContext
	defaultConnection *connection_deck.Connection
	enableCopy        bool
	handleOnSubmit    ConnDialogOnSubmitFunc
}

func NewConnectionForm(
	appCtx appcontext.AppContext,
	defaultConnection *connection_deck.Connection,
	enableCopy bool,
	handleOnSubmit ConnDialogOnSubmitFunc,
) *ConnectionForm {
	item := &ConnectionForm{
		appCtx:            appCtx,
		defaultConnection: defaultConnection,
		enableCopy:        enableCopy,
		handleOnSubmit:    handleOnSubmit,
	}
	item.ExtendBaseWidget(item)
	return item
}

func (w *ConnectionForm) CreateRenderer() fyne.WidgetRenderer {
	awsForm := w.buildAWSForm()
	awsForm.SubmitText = "Save"

	s3LikeForm := w.buildS3LikeForm()
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

	switch w.defaultConnection.Provider() {
	case connection_deck.AWSProvider:
		tabs.SelectIndex(0)
	case connection_deck.S3LikeProvider:
		tabs.SelectIndex(1)
	}
	return widget.NewSimpleRenderer(tabs)
}

func (w *ConnectionForm) AsDialog(label string) dialog.Dialog {
	d := dialog.NewCustom(label, "cancel", w, w.appCtx.Window())
	originalOnSubmit := w.handleOnSubmit
	w.handleOnSubmit = func(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) error {
		err := originalOnSubmit(name, accessKey, secretKey, bucket, options...)
		d.Hide()
		return err
	}
	d.Resize(fyne.NewSize(650, 200))
	return d
}

func (w *ConnectionForm) buildAWSForm() *widget.Form {
	// Init data bindings
	nameData := binding.NewString()
	nameData.Set(w.defaultConnection.Name())

	accessKeyData := binding.NewString()
	accessKeyData.Set(w.defaultConnection.AccessKey())

	secretKeyData := binding.NewString()
	secretKeyData.Set(w.defaultConnection.SecretKey())

	bucketData := binding.NewString()
	bucketData.Set(w.defaultConnection.Bucket())

	regionData := binding.NewString()
	regionData.Set(w.defaultConnection.Region())

	readOnlyData := binding.NewBool()
	readOnlyData.Set(w.defaultConnection.ReadOnly())

	// Create Form items
	nameFormItem := makeTextFormItemWithData(
		nameData,
		"Connection name",
		"My new connection",
		w.enableCopy,
		w.appCtx.Window(),
	)
	accessKeyFormItem := makeTextFormItemWithData(
		accessKeyData,
		"Access key Id",
		"Access key",
		w.enableCopy,
		w.appCtx.Window(),
	)
	secretKeyFormItem := makeTextFormItemWithData(
		secretKeyData,
		"Secret access key",
		"Secret key",
		w.enableCopy,
		w.appCtx.Window(),
	)

	bucketFormItem := makeTextFormItemWithData(
		bucketData,
		"Bucket name",
		"my-bucket",
		w.enableCopy,
		w.appCtx.Window(),
	)
	regionFormItem := makeTextFormItemWithData(
		regionData,
		"Region",
		"us-east-1",
		w.enableCopy,
		w.appCtx.Window(),
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
		if err := w.handleOnSubmit(
			uiutils.GetString(nameData),
			uiutils.GetString(accessKeyData),
			uiutils.GetString(secretKeyData),
			uiutils.GetString(bucketData),
			connection_deck.AsAWS(uiutils.GetString(regionData)),
			connection_deck.WithReadOnlyOption(uiutils.GetBool(readOnlyData)),
		); err == nil {
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

func (w *ConnectionForm) buildS3LikeForm() *widget.Form {
	// Init data bindings
	nameData := binding.NewString()
	nameData.Set(w.defaultConnection.Name())

	accessKeyData := binding.NewString()
	accessKeyData.Set(w.defaultConnection.AccessKey())

	secretKeyData := binding.NewString()
	secretKeyData.Set(w.defaultConnection.SecretKey())

	serverData := binding.NewString()
	serverData.Set(w.defaultConnection.Server())

	bucketData := binding.NewString()
	bucketData.Set(w.defaultConnection.Bucket())

	readOnlyData := binding.NewBool()
	readOnlyData.Set(w.defaultConnection.ReadOnly())

	useTlsData := binding.NewBool()
	useTlsData.Set(w.defaultConnection.IsTLSActivated())

	nameFormItem := makeTextFormItemWithData(
		nameData,
		"Connection name",
		"My new connection",
		w.enableCopy,
		w.appCtx.Window(),
	)
	accessKeyFormItem := makeTextFormItemWithData(
		accessKeyData,
		"Access key Id",
		"Access key",
		w.enableCopy,
		w.appCtx.Window(),
	)
	secretKeyFormItem := makeTextFormItemWithData(
		secretKeyData,
		"Secret access key",
		"Secret key",
		w.enableCopy,
		w.appCtx.Window(),
	)
	serverFormItem := makeTextFormItemWithData(
		serverData,
		"Server hostname",
		"s3.amazonaws.com",
		w.enableCopy,
		w.appCtx.Window(),
	)
	bucketFormItem := makeTextFormItemWithData(
		bucketData,
		"Bucket name",
		"my-bucket",
		w.enableCopy,
		w.appCtx.Window(),
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
		if err := w.handleOnSubmit(
			uiutils.GetString(nameData),
			uiutils.GetString(accessKeyData),
			uiutils.GetString(secretKeyData),
			uiutils.GetString(bucketData),
			connection_deck.AsS3Like(uiutils.GetString(serverData), uiutils.GetBool(useTlsData)),
			connection_deck.WithReadOnlyOption(uiutils.GetBool(readOnlyData)),
		); err == nil {
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
