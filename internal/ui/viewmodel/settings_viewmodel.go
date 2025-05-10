package viewmodel

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/settings"
	"github.com/thomas-marquis/s3-box/internal/utils"
)

const (
	settingsTimeout = 15 * time.Second
)

type SettingsViewModel interface {
	Save(s settings.Settings) error
	TimeoutInSeconds() binding.Int
	CurrentTimeout() time.Duration
	MaxFilePreviewSizeBytes() binding.Int
	CurrentColorTheme() settings.ColorTheme
	ChangeColorTheme(theme settings.ColorTheme) error
}

type settingsViewModelImpl struct {
	settingsRepo settings.Repository
	loading      binding.Bool
	errChan      chan error

	timeoutInSeconds binding.Int
	maxFilePreviewSizeBytes binding.Int
	fyneSettings     fyne.Settings
}

var _ SettingsViewModel = &settingsViewModelImpl{}

func NewSettingsViewModel(settingsRepo settings.Repository, fyneSettings fyne.Settings) SettingsViewModel {
	errChan := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()
	s, err := settingsRepo.Get(ctx)
	if err != nil {
		errChan <- fmt.Errorf("error getting settings: %w", err)
	}

	vm := &settingsViewModelImpl{
		settingsRepo:     settingsRepo,
		loading:          binding.NewBool(),
		errChan:          errChan,
		timeoutInSeconds: binding.NewInt(),
		maxFilePreviewSizeBytes: binding.NewInt(),
		fyneSettings:     fyneSettings,
	}

	vm.synchronize(s)

	go func() {
		for err := range errChan {
			fmt.Printf("Error in SettingsViewModel: %v\n", err)
		}
	}()

	return vm
}

func (vm *settingsViewModelImpl) Save(s settings.Settings) error {
	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()

	if err := vm.settingsRepo.Save(ctx, s); err != nil {
		vm.errChan <- fmt.Errorf("error saving settings: %w", err)
		return err
	}

	vm.synchronize(s)

	return nil
}

func (vm *settingsViewModelImpl) TimeoutInSeconds() binding.Int {
	return vm.timeoutInSeconds
}

func (vm *settingsViewModelImpl) CurrentTimeout() time.Duration {
	val, err := vm.timeoutInSeconds.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting timeout in seconds: %w", err)
		return settings.DefaultTimeoutInSeconds * time.Second
	}
	return time.Duration(val) * time.Second
}

func (vm *settingsViewModelImpl) MaxFilePreviewSizeBytes() binding.Int {
	return vm.maxFilePreviewSizeBytes
}

func (vm *settingsViewModelImpl) currentSettings() (settings.Settings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.CurrentTimeout())
	defer cancel()
	s, err := vm.settingsRepo.Get(ctx)
	if err != nil {
		return settings.Settings{}, fmt.Errorf("error getting settings: %w", err)
	}
	return s, nil
}

func (vm *settingsViewModelImpl) ChangeColorTheme(colorTheme settings.ColorTheme) error {
	vm.fyneSettings.SetTheme(utils.MapFyneColorTheme(colorTheme))

	newSettings, err := vm.currentSettings()
	if err != nil {
		return fmt.Errorf("error creating new settings: %w", err)
	}
	newSettings.Color = colorTheme

	ctx, cancel := context.WithTimeout(context.Background(), vm.CurrentTimeout())
	defer cancel()
	err = vm.settingsRepo.Save(ctx, newSettings)
	if err != nil {
		vm.errChan <- fmt.Errorf("error saving settings: %w", err)
	}

	return nil
}

func (vm *settingsViewModelImpl) CurrentColorTheme() settings.ColorTheme {
	s, err := vm.currentSettings()
	if err != nil {
		return settings.ColorThemeSystem
	}
	return s.Color
}

func (vm *settingsViewModelImpl) synchronize(s settings.Settings) {
	vm.timeoutInSeconds.Set(s.TimeoutInSeconds)
	vm.maxFilePreviewSizeBytes.Set(s.MaxFilePreviewSizeBytes)
}