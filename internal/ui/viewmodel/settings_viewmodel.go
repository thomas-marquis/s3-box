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
	if err := s.Register(
		settings.AString(values.SettingColorTheme, "white"),
		settings.AUint64(values.SettingEditFileSizeLimitByte, 20*1024),
		settings.ADuration(values.SettingTimeoutSec, 30*time.Second),
	); err != nil {
		panic(err)
	}

	// Set initial values from defaults and update bindings
	// The entity already has default values from the registration, so we just need to set the bindings
	if err := appState.Settings().TimeoutInSeconds().Set(int(30)); err != nil {
		panic(err)
	}
	if err := appState.Settings().EditorFileSizeLimitKB().Set(20); err != nil {
		panic(err)
	}
	if err := appState.Settings().ColorTheme().Set("white"); err != nil {
		panic(err)
	}

	s.Observe(values.SettingColorTheme, func(value any) {
		newTheme := value.(string)
		fyneSettings.SetTheme(apptheme.GetByName(newTheme))
		if err := appState.Settings().ColorTheme().Set(newTheme); err != nil {
			vm.notifier.NotifyError(err)
		}
	})

	s.Observe(values.SettingTimeoutSec, func(value any) {
		duration := value.(time.Duration)
		if err := appState.Settings().TimeoutInSeconds().Set(int(duration.Seconds())); err != nil {
			vm.notifier.NotifyError(err)
		}
	})

	s.Observe(values.SettingEditFileSizeLimitByte, func(value any) {
		bytes := value.(uint64)
		// Convert bytes to KB
		kb := int(bytes / 1024)
		if err := appState.Settings().EditorFileSizeLimitKB().Set(kb); err != nil {
			vm.notifier.NotifyError(err)
		}
	})

	// Load settings from storage
	// We call Load() to trigger loading from persistent storage
	// The LoadSucceeded event will be handled asynchronously
	evt, err := s.Load()
	if err != nil {
		panic(err)
	}
	eventBus.Publish(evt)

	return vm
}

func (v *settingsViewModelImpl) Save() {
	s := v.state.Settings().Get()

	// Check if the entity is ready to accept writes
	// If it's in LoadingState, we need to wait for Load to complete
	// For now, we just return an error if not ready
	if !s.State().CanWrite() {
		v.notifier.NotifyError(settings.ErrNotReady)
		return
	}

	// Read current values from bindings and write to settings
	if timeoutVal, err := v.state.Settings().TimeoutInSeconds().Get(); err == nil {
		if writeErr := s.Write(values.SettingTimeoutSec, time.Duration(timeoutVal)*time.Second); writeErr != nil {
			v.notifier.NotifyError(writeErr)
			return
		}
	}

	if fileLimitVal, err := v.state.Settings().EditorFileSizeLimitKB().Get(); err == nil {
		// Convert KB to bytes
		if writeErr := s.Write(values.SettingEditFileSizeLimitByte, uint64(fileLimitVal*1024)); writeErr != nil {
			v.notifier.NotifyError(writeErr)
			return
		}
	}

	if colorThemeVal, err := v.state.Settings().ColorTheme().Get(); err == nil {
		if writeErr := s.Write(values.SettingColorTheme, colorThemeVal); writeErr != nil {
			v.notifier.NotifyError(writeErr)
			return
		}
	}

	// Now trigger the save
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
