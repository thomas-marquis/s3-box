package viewmodel

import (
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
	loading      binding.Bool
	notifier     notification.Repository
	fyneSettings fyne.Settings
	state        *state.State
	bus          event.Bus
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

	// Observe color theme changes to update Fyne theme
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
	s := v.state.Settings().Get()

	evt, err := s.Save()
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.bus.Publish(evt)
}

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
