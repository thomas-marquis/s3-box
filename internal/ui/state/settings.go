package state

import (
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
	"github.com/thomas-marquis/s3-box/internal/ui/values"
)

type SettingsState struct {
	aggregate *settings.Settings

	timeout    binding.Int
	fileLimit  binding.Int
	colorTheme binding.String
}

func newSettingsState() *SettingsState {
	settingsAgg := settings.NewSettings()
	if err := settingsAgg.Register(
		settings.AString(values.SettingColorTheme, values.ColorThemeSystem),
		settings.AUint64(values.SettingEditFileSizeLimitByte, 20*1024),
		settings.ADuration(values.SettingTimeoutSec, 30*time.Second),
	); err != nil {
		panic(err)
	}

	return &SettingsState{
		aggregate:  settingsAgg,
		timeout:    uiutils.NewSettingsBindingIntForDuration(settingsAgg, values.SettingTimeoutSec),
		fileLimit:  uiutils.NewSettingsBindingIntToUint64KB(settingsAgg, values.SettingEditFileSizeLimitByte),
		colorTheme: uiutils.NewSettingsBindingString(settingsAgg, values.SettingColorTheme),
	}
}

func (s *SettingsState) Get() *settings.Settings {
	return s.aggregate
}

func (s *SettingsState) TimeoutInSeconds() binding.Int {
	return s.timeout
}

func (s *SettingsState) EditorFileSizeLimitKB() binding.Int {
	return s.fileLimit
}

func (s *SettingsState) ColorTheme() binding.String {
	return s.colorTheme
}

func (s *SettingsState) CurrentTimeout() time.Duration {
	if s.aggregate.IsExistsWithType(values.SettingTimeoutSec, settings.DurationType) {
		return s.aggregate.ReadDuration(values.SettingTimeoutSec)
	}
	return 30 * time.Second
}
