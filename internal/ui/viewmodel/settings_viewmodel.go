package viewmodel

import (
	"context"
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	"github.com/thomas-marquis/s3-box/internal/utils"
)

const (
	settingsTimeout = 15 * time.Second
)

type SettingsViewModel interface {
	Save() error
	TimeoutInSeconds() binding.Int
	CurrentTimeout() time.Duration
	MaxFilePreviewSizeBytes() binding.Int
	CurrentMaxFilePreviewSizeBytes() int
	ColorTheme() binding.String
}

type settingsViewModelImpl struct {
	settingsRepo settings.Repository
	loading      binding.Bool
	errorStream  chan<- error

	timeoutInSeconds        binding.Int
	maxFilePreviewSizeBytes binding.Int
	fyneSettings            fyne.Settings
	colorTheme              binding.String
}

func NewSettingsViewModel(settingsRepo settings.Repository, fyneSettings fyne.Settings, errorStream chan<- error) SettingsViewModel {
	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()
	s, err := settingsRepo.Get(ctx)
	if err != nil {
		errorStream <- fmt.Errorf("error getting settings: %w", err)
	}
	fyneSettings.SetTheme(utils.MapFyneColorTheme(s.Color))

	themeBinding := binding.NewString()

	themeBinding.AddListener(binding.NewDataListener(func() {
		currThemeName, err := themeBinding.Get()
		if err != nil {
			errorStream <- fmt.Errorf("error getting color theme: %w", err)
			return
		}
		currTheme, err := settings.NewColorThemeFromString(currThemeName)
		if err != nil {
			errorStream <- fmt.Errorf("error converting color theme: %w", err)
			return
		}
		fyneSettings.SetTheme(utils.MapFyneColorTheme(currTheme))
	}))

	vm := &settingsViewModelImpl{
		settingsRepo:            settingsRepo,
		loading:                 binding.NewBool(),
		errorStream:             errorStream,
		timeoutInSeconds:        binding.NewInt(),
		maxFilePreviewSizeBytes: binding.NewInt(),
		fyneSettings:            fyneSettings,
		colorTheme:              themeBinding,
	}

	vm.synchronize(s)

	return vm
}

func (vm *settingsViewModelImpl) Save() error {
	timeout, err := vm.timeoutInSeconds.Get()
	if err != nil {
		vm.errorStream <- fmt.Errorf("error getting timeout in seconds: %w", err)
		return err
	}
	maxFilePreviewSizeMegaBytes, err := vm.maxFilePreviewSizeBytes.Get()
	if err != nil {
		vm.errorStream <- fmt.Errorf("error getting max file preview size in mega bytes: %w", err)
		return err
	}
	colorThemeString, err := vm.colorTheme.Get()
	if err != nil {
		vm.errorStream <- fmt.Errorf("error getting color theme: %w", err)
		return err
	}
	colorTheme, err := settings.NewColorThemeFromString(colorThemeString)
	if err != nil {
		vm.errorStream <- fmt.Errorf("error converting color theme: %w", err)
		return err
	}

	s, err := settings.NewSettings(timeout, utils.MegaToBytes(maxFilePreviewSizeMegaBytes))
	if err != nil {
		vm.errorStream <- fmt.Errorf("error creating settings: %w", err)
		return err
	}
	s.Color = colorTheme

	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()

	if err := vm.settingsRepo.Save(ctx, s); err != nil {
		vm.errorStream <- fmt.Errorf("error saving settings: %w", err)
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
		vm.errorStream <- fmt.Errorf("error getting timeout in seconds: %w", err)
		return settings.DefaultTimeoutInSeconds * time.Second
	}
	return time.Duration(val) * time.Second
}

func (vm *settingsViewModelImpl) MaxFilePreviewSizeBytes() binding.Int {
	return vm.maxFilePreviewSizeBytes
}

func (vm *settingsViewModelImpl) CurrentMaxFilePreviewSizeBytes() int {
	val, err := vm.maxFilePreviewSizeBytes.Get()
	if err != nil {
		vm.errorStream <- fmt.Errorf("error getting max file preview size in mega bytes: %w", err)
		return settings.DefaultMaxFilePreviewSizeBytes
	}
	return val
}

func (vm *settingsViewModelImpl) ColorTheme() binding.String {
	return vm.colorTheme
}

func (vm *settingsViewModelImpl) synchronize(s settings.Settings) {
	vm.timeoutInSeconds.Set(s.TimeoutInSeconds)
	if s.MaxFilePreviewSizeBytes > math.MaxInt {
		vm.errorStream <- fmt.Errorf("max file preview size exceeds maximum allowed value: clamping to max int")
		vm.maxFilePreviewSizeBytes.Set(math.MaxInt)
	} else {
		vm.maxFilePreviewSizeBytes.Set(s.MaxFilePreviewSizeBytes)
	}
	vm.colorTheme.Set(s.Color.String())
}
