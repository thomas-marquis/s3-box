package viewmodel

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"

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
	FileSizeLimitKB() binding.Int
	CurrentFileSizeLimitBytes() int
	ColorTheme() binding.String
}

type settingsViewModelImpl struct {
	settingsRepo settings.Repository
	loading      binding.Bool
	notifier     notification.Repository

	timeoutInSeconds binding.Int
	fileSizeLimitKB  binding.Int
	fyneSettings     fyne.Settings
	colorTheme       binding.String
}

func NewSettingsViewModel(
	settingsRepo settings.Repository,
	fyneSettings fyne.Settings,
	notifier notification.Repository,
) SettingsViewModel {
	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()
	s, err := settingsRepo.Get(ctx)
	if err != nil {
		notifier.NotifyError(fmt.Errorf("error getting settings: %w", err))
	}
	fyneSettings.SetTheme(apptheme.GetByName(s.Color))
	themeBinding := binding.NewString()

	themeBinding.AddListener(binding.NewDataListener(func() {
		currThemeName, err := themeBinding.Get()
		if err != nil {
			notifier.NotifyError(fmt.Errorf("error getting color theme: %w", err))
			return
		}
		currTheme, err := settings.NewColorThemeFromString(currThemeName)
		if err != nil {
			notifier.NotifyError(fmt.Errorf("error converting color theme: %w", err))
			return
		}
		fyneSettings.SetTheme(apptheme.GetByName(currTheme))
	}))

	vm := &settingsViewModelImpl{
		settingsRepo:     settingsRepo,
		loading:          binding.NewBool(),
		notifier:         notifier,
		timeoutInSeconds: binding.NewInt(),
		fileSizeLimitKB:  binding.NewInt(),
		fyneSettings:     fyneSettings,
		colorTheme:       themeBinding,
	}

	vm.synchronize(s)

	return vm
}

func (vm *settingsViewModelImpl) Save() error {
	timeout, err := vm.timeoutInSeconds.Get()
	if err != nil {
		wErr := fmt.Errorf("error getting timeout in seconds: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
	}
	maxFilePreviewSizeKB, err := vm.fileSizeLimitKB.Get()
	if err != nil {
		wErr := fmt.Errorf("error getting max file preview size in mega bytes: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
	}
	colorThemeString, err := vm.colorTheme.Get()
	if err != nil {
		wErr := fmt.Errorf("error getting color theme: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
	}
	colorTheme, err := settings.NewColorThemeFromString(colorThemeString)
	if err != nil {
		wErr := fmt.Errorf("error converting color theme: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
	}

	s, err := settings.NewSettings(timeout, utils.KBToBytes(maxFilePreviewSizeKB))
	if err != nil {
		wErr := fmt.Errorf("error creating settings: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
	}
	s.Color = colorTheme

	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()

	if err := vm.settingsRepo.Save(ctx, s); err != nil {
		wErr := fmt.Errorf("error saving settings: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
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
		vm.notifier.NotifyError(fmt.Errorf("error getting timeout in seconds: %w", err))
		return settings.DefaultTimeoutInSeconds * time.Second
	}
	return time.Duration(val) * time.Second
}

func (vm *settingsViewModelImpl) FileSizeLimitKB() binding.Int {
	return vm.fileSizeLimitKB
}

func (vm *settingsViewModelImpl) CurrentFileSizeLimitBytes() int {
	val, err := vm.fileSizeLimitKB.Get()
	if err != nil {
		vm.notifier.NotifyError(fmt.Errorf("error getting file preview/edit size limit: %w", err))
		return settings.DefaultMaxFilePreviewSizeBytes
	}
	return utils.KBToBytes(val)
}

func (vm *settingsViewModelImpl) ColorTheme() binding.String {
	return vm.colorTheme
}

func (vm *settingsViewModelImpl) synchronize(s settings.Settings) {
	vm.timeoutInSeconds.Set(s.TimeoutInSeconds) //nolint:errcheck
	if s.MaxFilePreviewSizeBytes > math.MaxInt {
		vm.notifier.NotifyError(
			fmt.Errorf("max file preview size exceeds maximum allowed value: clamping to max int"))
		vm.fileSizeLimitKB.Set(math.MaxInt) //nolint:errcheck
	} else {
		vm.fileSizeLimitKB.Set(utils.BytesToKB(s.MaxFilePreviewSizeBytes)) //nolint:errcheck
	}
	vm.colorTheme.Set(s.Color.String()) //nolint:errcheck
}
