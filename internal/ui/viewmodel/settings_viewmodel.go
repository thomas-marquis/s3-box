package viewmodel

import (
	"time"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
	"github.com/thomas-marquis/s3-box/internal/ui/values"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

type SettingsViewModel interface {
	Save()
	Cancel()
	SaveEditorSelectors()
}

type settingsViewModelImpl struct {
	notifier              notification.Repository
	fyneSettings          fyne.Settings
	fynePrefs             fyne.Preferences
	state                 *state.State
	bus                   event.Bus
	editorMappingsRepository editor.MappingRepository
}

func NewSettingsViewModel(
	fyneSettings fyne.Settings,
	fynePrefs fyne.Preferences,
	notifier notification.Repository,
	appState *state.State,
	bus event.Bus,
	editorMappingsRepository editor.MappingRepository,
) SettingsViewModel {
	vm := &settingsViewModelImpl{
		notifier:                 notifier,
		fyneSettings:             fyneSettings,
		fynePrefs:                fynePrefs,
		state:                    appState,
		bus:                      bus,
		editorMappingsRepository: editorMappingsRepository,
	}

	s := appState.Settings().Get()

	bus.Subscribe().
		On(event.IsOneOf(
			settings.LoadFailedType, settings.LoadSucceededType,
			settings.SaveSucceededType, settings.SaveFailedType,
		), func(e event.Event) {
			u.Skip(appState.Settings().IsReady().Set(true))
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

	vm.loadEditorSelectors()

	return vm
}

func (v *settingsViewModelImpl) Save() {
	s := v.state.Settings().Get()

	if !s.HasPendingEvents() {
		u.Skip(v.state.Settings().StatusMessage().Set("No changes to save"))
		go func() {
			time.Sleep(2 * time.Second)
			u.Skip(v.state.Settings().StatusMessage().Set(""))
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

// SaveEditorSelectors saves editor selectors to preferences and updates the state
func (v *settingsViewModelImpl) SaveEditorSelectors() {
	selectors := u.SkipV(v.state.Settings().EditorSelectors().Get())

	mappings := make([]editor.Mapping, 0, len(selectors))
	for _, selector := range selectors {
		for _, mapping := range selector.Mappings() {
			mappings = append(mappings, mapping)
		}
	}

	if err := v.editorMappingsRepository.SaveAll(mappings); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	u.Skip(v.state.Settings().StatusMessage().Set("New settings saved"))
}

func (v *settingsViewModelImpl) updateStatusMessage(evt event.Event) {
	switch evt.Type() {
	case settings.LoadSucceededType:
		v.state.Settings().SyncStatusMessage()
	case settings.SaveSucceededType:
		v.state.Settings().SyncStatusMessage()
	case settings.LoadFailedType:
		u.Skip(v.state.Settings().StatusMessage().Set("Loading error"))
	case settings.SaveFailedType:
		u.Skip(v.state.Settings().StatusMessage().Set("Saving error"))
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

func (v *settingsViewModelImpl) loadEditorSelectors() []*editor.Selector {
	if v.state.Settings().EditorSelectors().Length() > 0 {
		return u.SkipV(v.state.Settings().EditorSelectors().Get())
	}

	var selectors []*editor.Selector
	storedJSON := v.fynePrefs.String(values.SettingFileSelectors)
	if storedJSON == "" {
		selectors = initSelectors()
		v.state.Settings().EditorSelectors().Set(selectors)
		return selectors
	}

	if err := json.Unmarshal([]byte(storedJSON), &selectors); err != nil {
		selectors = initSelectors()
		v.state.Settings().EditorSelectors().Set(selectors)
		return selectors
	}

	if len(selectors) == 0 {
		selectors = initSelectors()
	}

	u.Skip(v.state.Settings().EditorSelectors().Set(selectors))
	return selectors
}
