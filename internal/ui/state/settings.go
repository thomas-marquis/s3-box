package state

import (
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

type SettingsState struct {
	aggregate *settings.SettingsV3
}

func (s *SettingsState) Get() *settings.SettingsV3 {
	return s.aggregate
}
