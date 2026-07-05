package state

import (
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	"github.com/thomas-marquis/s3-box/internal/ui/values"
)

type SettingsState struct {
	aggregate *settings.Settings

	timeout    binding.Int
	fileLimit  binding.Int
	colorTheme binding.String
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
