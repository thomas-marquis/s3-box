package viewmodel

import (
	"time"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/csveditor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	"github.com/thomas-marquis/s3-box/internal/ui/values"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type SettingsViewModel interface {
	Save()
	Cancel()
	SaveEditorSelectors()
}

type settingsViewModelImpl struct {
	notifier                 notification.Repository
	fyneSettings             fyne.Settings
	fynePrefs                fyne.Preferences
	state                    *state.State
	bus                      event.Bus
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

	vm.loadEditorMappings()

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
	selector := v.state.Editors().Selector()
	mappings := selector.Mappings()

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

func (v *settingsViewModelImpl) loadEditorMappings() {
	selector := v.state.Editors().Selector()

	defaultEditors := []editor.Factory{
		&texteditor.Factory{},
		&csveditor.Factory{},
	}

	for _, factory := range defaultEditors {
		selector.RegisterEditor(factory)
	}

	mappings, err := v.editorMappingsRepository.GetAll()
	if err != nil {
		return
	}

	if len(mappings) == 0 {
		for _, factory := range defaultEditors {
			mappings = append(mappings, editor.Mapping{
				RegexpPattern: factory.DefaultFileRegexpPattern(),
				EditorName:    factory.Name(),
			})
		}

		if saveErr := v.editorMappingsRepository.SaveAll(mappings); saveErr != nil {
			return
		}
	}

	for _, mapping := range mappings {
		u.Skip(selector.RegisterMapping(mapping.EditorName, mapping.RegexpPattern))
	}
}
