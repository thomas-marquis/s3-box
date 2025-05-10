package viewmodel

import (
	"context"
	"fmt"
	"math"
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
	Save() error
	TimeoutInSeconds() binding.Int
	CurrentTimeout() time.Duration
	MaxFilePreviewSizeMegaBytes() binding.Int
	ColorTheme() binding.String
}

type settingsViewModelImpl struct {
	settingsRepo settings.Repository
	loading      binding.Bool
	errChan      chan error

	timeoutInSeconds binding.Int
	maxFilePreviewSizeMegaBytes binding.Int
	fyneSettings     fyne.Settings
	colorTheme binding.String
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

	themeBinding := binding.NewString()
	themeBinding.AddListener(binding.NewDataListener(func() {
		currThemeName, err := themeBinding.Get()
		if err != nil {
			errChan <- fmt.Errorf("error getting color theme: %w", err)
			return
		}
		currTheme, err := settings.NewColorThemeFromString(currThemeName)
		if err != nil {
			errChan <- fmt.Errorf("error converting color theme: %w", err)
			return
		}
		fyneSettings.SetTheme(utils.MapFyneColorTheme(currTheme))
	}))

	vm := &settingsViewModelImpl{
		settingsRepo:     settingsRepo,
		loading:          binding.NewBool(),
		errChan:          errChan,
		timeoutInSeconds: binding.NewInt(),
		maxFilePreviewSizeMegaBytes: binding.NewInt(),
		fyneSettings:     fyneSettings,
		colorTheme:       themeBinding,
	}

	vm.synchronize(s)

	go func() {
		for err := range errChan {
			fmt.Printf("Error in SettingsViewModel: %v\n", err)
		}
	}()

	return vm
}

func (vm *settingsViewModelImpl) Save() error {
	timeout, err := vm.timeoutInSeconds.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting timeout in seconds: %w", err)
		return err
	}
	maxFilePreviewSizeMegaBytes, err := vm.maxFilePreviewSizeMegaBytes.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting max file preview size in mega bytes: %w", err)
		return err
	}
	colorThemeString, err := vm.colorTheme.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting color theme: %w", err)
		return err
	}
	colorTheme, err := settings.NewColorThemeFromString(colorThemeString)
	if err != nil {
		vm.errChan <- fmt.Errorf("error converting color theme: %w", err)
		return err
	}

	s, err := settings.NewSettings(timeout, utils.MegaToBytes(int64(maxFilePreviewSizeMegaBytes)))
	if err != nil {
		vm.errChan <- fmt.Errorf("error creating settings: %w", err)
		return err
	}
	s.Color = colorTheme

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

func (vm *settingsViewModelImpl) MaxFilePreviewSizeMegaBytes() binding.Int {
	return vm.maxFilePreviewSizeMegaBytes
}

func (vm *settingsViewModelImpl) ColorTheme() binding.String {
	return vm.colorTheme
}

func (vm *settingsViewModelImpl) synchronize(s settings.Settings) {
	vm.timeoutInSeconds.Set(s.TimeoutInSeconds)
	if s.MaxFilePreviewSizeBytes > math.MaxInt {
		vm.errChan <- fmt.Errorf("max file preview size exceeds maximum allowed value: clamping to max int")
		vm.maxFilePreviewSizeMegaBytes.Set(math.MaxInt)
	} else {
		vm.maxFilePreviewSizeMegaBytes.Set(int(utils.BytesToMB(s.MaxFilePreviewSizeBytes)))
	}
	vm.colorTheme.Set(s.Color.String())
}
