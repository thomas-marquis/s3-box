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

	timeout    binding.Item[time.Duration]
	fileLimit  binding.Item[uint64]
	colorTheme binding.String

	isReady       binding.Bool
	statusMessage binding.String
}

func newSettingsState() *SettingsState {
	settingsAgg := settings.NewSettings()
	if err := settingsAgg.Register(
		settings.AString(values.SettingColorTheme, values.DefaultColorTheme),
		settings.AUint64(values.SettingEditFileSizeLimitByte, values.DefaultMaxFileSizeEditBytes),
		settings.ADuration(values.SettingTimeoutSec, values.DefaultTimeout),
	); err != nil {
		panic(err)
	}

	state := &SettingsState{
		aggregate:     settingsAgg,
		timeout:       uiutils.NewSettingsBindingDuration(settingsAgg, values.SettingTimeoutSec),
		fileLimit:     uiutils.NewSettingsBindingIntToUint64(settingsAgg, values.SettingEditFileSizeLimitByte),
		colorTheme:    uiutils.NewSettingsBindingString(settingsAgg, values.SettingColorTheme),
		isReady:       binding.NewBool(),
		statusMessage: binding.NewString(),
	}

	state.SyncStatusMessage()

	return state
}

func (s *SettingsState) Get() *settings.Settings {
	return s.aggregate
}

func (s *SettingsState) Timeout() binding.Item[time.Duration] {
	return s.timeout
}

func (s *SettingsState) TimeoutValue() time.Duration {
	val, err := s.timeout.Get()
	if err != nil {
		logger.Printf("Error reading timeout value from state: %s. Falling back to default value", err)
		return values.DefaultTimeout
	}
	return val
}

func (s *SettingsState) EditorFileSizeLimitBytes() binding.Item[uint64] {
	return s.fileLimit
}

func (s *SettingsState) EditorFileSizeLimitBytesValue() uint64 {
	val, err := s.fileLimit.Get()
	if err != nil {
		logger.Printf("Error reading file size limit from state: %s. Falling back to default value", err)
		return values.DefaultMaxFileSizeEditBytes
	}
	return val
}

func (s *SettingsState) ColorTheme() binding.String {
	return s.colorTheme
}

func (s *SettingsState) IsReady() binding.Bool {
	return s.isReady
}

func (s *SettingsState) StatusMessage() binding.String {
	return s.statusMessage
}

func (s *SettingsState) SyncStatusMessage() {
	state := s.aggregate.State()
	switch state.String() {
	case "loading":
		s.statusMessage.Set("Loading...") //nolint:errcheck
	case "saving":
		s.statusMessage.Set("Saving...") //nolint:errcheck
	default:
		s.statusMessage.Set("") //nolint:errcheck
	}
}
