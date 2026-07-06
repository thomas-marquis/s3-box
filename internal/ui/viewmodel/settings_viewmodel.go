package viewmodel

import (
	"time"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
	"github.com/thomas-marquis/s3-box/internal/ui/values"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

type SettingsViewModel interface {
	Save()
	Cancel()
}

type settingsViewModelImpl struct {
	notifier     notification.Repository
	fyneSettings fyne.Settings
	state        *state.State
	bus          event.Bus
}

func NewSettingsViewModel(
	fyneSettings fyne.Settings,
	notifier notification.Repository,
	appState *state.State,
	bus event.Bus,
) SettingsViewModel {
	vm := &settingsViewModelImpl{
		notifier:     notifier,
		fyneSettings: fyneSettings,
		state:        appState,
		bus:          bus,
	}

	s := appState.Settings().Get()

	bus.Subscribe().
		On(event.IsOneOf(
			settings.LoadFailedType, settings.LoadSucceededType,
			settings.SaveSucceededType, settings.SaveFailedType,
		), func(e event.Event) {
			appState.Settings().IsReady().Set(true) //nolint:errcheck
			vm.notifySettings(e)
			vm.updateStatusMessage(e)
		}).
		On(event.Is(settings.WriteSucceededType), vm.notifySettings).
		On(event.IsOneOf(settings.SaveFailedType, settings.LoadFailedType), vm.handleFailure).
		On(event.Is(settings.SaveSucceededType), func(e event.Event) {
			fyne.CurrentApp().SendNotification(fyne.NewNotification("Settings saved", ""))
			appState.Settings().SyncStatusMessage()
		}).
		On(event.Is(settings.LoadTriggeredType), func(e event.Event) {
			appState.Settings().SyncStatusMessage()
		}).
		ListenWithWorkers(1)

	s.Observe(values.SettingColorTheme, func(value any) {
		newTheme := value.(string)
		fyne.Do(func() {
			fyneSettings.SetTheme(apptheme.GetByName(newTheme))
		})
	})

	evt, err := s.Load()
	if err != nil {
		panic(err)
	}
	bus.Publish(evt)

	return vm
}

func (v *settingsViewModelImpl) Save() {
	s := v.state.Settings().Get()

	if !s.HasPendingEvents() {
		v.state.Settings().StatusMessage().Set("No changes to save") //nolint:errcheck
		go func() {
			time.Sleep(2 * time.Second)
			v.state.Settings().StatusMessage().Set("") //nolint:errcheck
		}()
		return
	}

	evt, err := s.Save()
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.bus.Publish(evt)
	v.state.Settings().SyncStatusMessage()
}

func (v *settingsViewModelImpl) Cancel() {
	s := v.state.Settings().Get()
	s.Cancel()
	v.state.Settings().SyncStatusMessage()
}

func (v *settingsViewModelImpl) updateStatusMessage(evt event.Event) {
	switch evt.Type() {
	case settings.LoadSucceededType:
		v.state.Settings().SyncStatusMessage()
	case settings.SaveSucceededType:
		v.state.Settings().SyncStatusMessage()
	case settings.LoadFailedType:
		v.state.Settings().StatusMessage().Set("Loading error") //nolint:errcheck
	case settings.SaveFailedType:
		v.state.Settings().StatusMessage().Set("Saving error") //nolint:errcheck
	}
}

func (v *settingsViewModelImpl) notifySettings(evt event.Event) {
	if err := v.state.Settings().Get().Notify(evt); err != nil {
		v.notifier.NotifyError(err)
	}
}

func (v *settingsViewModelImpl) handleFailure(evt event.Event) {
	var err error
	switch pl := evt.Payload().(type) {
	case settings.LoadFailed:
		err = pl.Err
	case settings.SaveFailed:
		err = pl.Err
	}
	v.notifier.NotifyError(err)
	fyne.CurrentApp().SendNotification(fyne.NewNotification("Ooops...", err.Error()))
}
