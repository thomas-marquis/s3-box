package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/settings"

	"fyne.io/fyne/v2"
)

const settingsKey = "settings"

type SettingRepositoryImpl struct {
	prefs fyne.Preferences
}

func NewSettingsRepository(prefs fyne.Preferences) settings.Repository {
	return &SettingRepositoryImpl{prefs: prefs}
}

func (r *SettingRepositoryImpl) Save(ctx context.Context, s settings.Settings) error {
	settingJson, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("Save: %w", err)
	}
	r.prefs.SetString(settingsKey, string(settingJson))
	return nil
}

func (r *SettingRepositoryImpl) Get(ctx context.Context) (settings.Settings, error) {
	content := r.prefs.String(settingsKey)
	if content == "" || content == "null" {
		return settings.DefaultSettings(), nil
	}

	s, err := fromJson[settings.Settings](content)
	if err != nil {
		return settings.Settings{}, fmt.Errorf("Get: %w", err)
	}
	return s, nil
}