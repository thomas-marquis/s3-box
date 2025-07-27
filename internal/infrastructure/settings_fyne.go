package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/settings"

	"fyne.io/fyne/v2"
)

const settingsKey = "settings"

type settingsDTO struct {
	TimeoutInSeconds        int    `json:"timeoutInSeconds"`
	MaxFilePreviewSizeBytes int    `json:"maxFilePreviewSizeBytes"`
	ColorTheme              string `json:"colorTheme"`
}

func (s *settingsDTO) toSettings() settings.Settings {
	entity, err := settings.NewSettings(s.TimeoutInSeconds, s.MaxFilePreviewSizeBytes)
	if err != nil {
		logger.Printf("error converting settings: %v", err)
		return settings.DefaultSettings()
	}
	entity.Color, err = settings.NewColorThemeFromString(s.ColorTheme)
	if err != nil {
		logger.Printf("error converting color theme: %v", err)
		return settings.DefaultSettings()
	}
	return entity
}

func newSettingsDTO(s settings.Settings) *settingsDTO {
	return &settingsDTO{
		TimeoutInSeconds:        s.TimeoutInSeconds,
		MaxFilePreviewSizeBytes: s.MaxFilePreviewSizeBytes,
		ColorTheme:              s.Color.String(),
	}
}

type SettingRepositoryImpl struct {
	prefs fyne.Preferences
}

func NewSettingsRepository(prefs fyne.Preferences) settings.Repository {
	return &SettingRepositoryImpl{prefs: prefs}
}

func (r *SettingRepositoryImpl) Save(ctx context.Context, s settings.Settings) error {
	dto := newSettingsDTO(s)
	settingJson, err := json.Marshal(dto)
	if err != nil {
		logger.Printf("error marshalling settings: %v", err)
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

	dto, err := fromJson[settingsDTO](content)
	if err != nil {
		logger.Printf("error converting settings: %v", err)
		return settings.DefaultSettings(), nil
	}
	s := dto.toSettings()
	return s, nil
}
