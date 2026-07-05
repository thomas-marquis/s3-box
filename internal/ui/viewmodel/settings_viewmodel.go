package viewmodel

import (
	"time"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
	"github.com/thomas-marquis/s3-box/internal/ui/values"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

type SettingsViewModel interface {
	Save()
}

type settingsViewModelImpl struct {
	loading  binding.Bool
	notifier notification.Repository

	fyneSettings fyne.Settings

	state *state.State
	bus   event.Bus
}

func NewSettingsViewModel(
	fyneSettings fyne.Settings,
	notifier notification.Repository,
	appState *state.State,
	eventBus event.Bus,
) SettingsViewModel {
	vm := &settingsViewModelImpl{
		loading:      binding.NewBool(),
		notifier:     notifier,
		fyneSettings: fyneSettings,
		state:        appState,
		bus:          eventBus,
	}

	eventBus.Subscribe().
		On(event.IsOneOf(settings.LoadSucceededType, settings.SaveSucceededType), vm.notifySettings).
		On(event.IsOneOf(settings.SaveFailedType, settings.LoadFailedType), vm.handleFailure).
		ListenWithWorkers(1)

	s := appState.Settings().Get()
	if err := s.Register(
		settings.AString(values.SettingColorTheme, "white"),
		settings.AUint64(values.SettingEditFileSizeLimitByte, 20*1024),
		settings.ADuration(values.SettingTimeoutSec, 30*time.Second),
	); err != nil {
		panic(err)
	}

	s.Observe(values.SettingColorTheme, func(value any) {
		newTheme := value.(string)
		fyneSettings.SetTheme(apptheme.GetByName(newTheme))
	})

	evt, err := s.Load()
	if err != nil {
		panic(err)
	}
	eventBus.Publish(evt)

	return vm
}

func (v *settingsViewModelImpl) Save() {
	v.bus.Publish(v.state.Settings().Get().Save())
}

//
//func (v *settingsViewModelImpl) TimeoutInSeconds() binding.Int {
//	return v.timeoutInSeconds
//}
//
//func (v *settingsViewModelImpl) CurrentTimeout() time.Duration {
//	val, err := v.timeoutInSeconds.Get()
//	if err != nil {
//		v.notifier.NotifyError(fmt.Errorf("error getting timeout in seconds: %w", err))
//		return settings.DefaultTimeoutInSeconds * time.Second
//	}
//	return time.Duration(val) * time.Second
//}
//
//func (v *settingsViewModelImpl) FileSizeLimitKB() binding.Int {
//	return v.fileSizeLimitKB
//}
//
//func (v *settingsViewModelImpl) CurrentFileSizeLimitBytes() int {
//	val, err := v.fileSizeLimitKB.Get()
//	if err != nil {
//		v.notifier.NotifyError(fmt.Errorf("error getting file preview/edit size limit: %w", err))
//		return settings.DefaultMaxFilePreviewSizeBytes
//	}
//	return utils.KBToBytes(val)
//}

//func (v *settingsViewModelImpl) synchronize(s settings.Settings) {
//	ignoreErr := func(err error) {
//		v.notifier.NotifyError(err)
//	}
//
//	ignoreErr(v.state.Settings().TimeoutSec().Set(s.TimeoutInSeconds))
//
//	if s.MaxFilePreviewSizeBytes > math.MaxInt {
//		v.notifier.NotifyError(
//			fmt.Errorf("max file preview size exceeds maximum allowed value: clamping to max int"))
//		ignoreErr(v.state.Settings().EditorFileSizeLimitKB().Set(math.MaxInt))
//	} else {
//		ignoreErr(v.state.Settings().EditorFileSizeLimitKB().Set(
//			utils.BytesToKB(s.MaxFilePreviewSizeBytes)))
//	}
//
//	ignoreErr(v.state.Settings().ColorTheme().Set(s.Color.String()))
//}

func (v *settingsViewModelImpl) notifySettings(evt event.Event) {
	if err := v.state.Settings().Get().Notify(evt); err != nil {
		v.notifier.NotifyError(err)
	}
}

func (v *settingsViewModelImpl) handleFailure(evt event.Event) {
	v.notifySettings(evt)
	var err error
	switch pl := evt.Payload().(type) {
	case settings.LoadFailed:
		err = pl.Err
	case settings.SaveFailed:
		err = pl.Err
	}
	v.notifier.NotifyError(err)
}
