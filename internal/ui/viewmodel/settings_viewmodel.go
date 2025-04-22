package viewmodel

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/settings"
)

const (
	settingsTimeout = 15 * time.Second
)


type SettingsViewModel interface {
	Save(s settings.Settings) error
	TimeoutInSeconds() time.Duration
}

type settingsViewModelImpl struct {
	settingsRepo settings.Repository
	loading      binding.Bool
	errChan      chan error

	timeoutInSeconds binding.Int
}

func NewSettingsViewModel(settingsRepo settings.Repository) SettingsViewModel {
	errChan := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()
	s, err := settingsRepo.Get(ctx)
	if err != nil {
		errChan <- fmt.Errorf("error getting settings: %w", err)
	}

	vm := &settingsViewModelImpl{
		settingsRepo: settingsRepo,
		loading:      binding.NewBool(),
		errChan:      errChan,
		timeoutInSeconds: binding.NewInt(),
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

func (vm *settingsViewModelImpl) TimeoutInSeconds() time.Duration {
	val, err := vm.timeoutInSeconds.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting timeout in seconds: %w", err)
		return settings.DefaultTimeoutInSeconds * time.Second
	}
	return time.Duration(val) * time.Second
}

func (vm *settingsViewModelImpl) synchronize(s settings.Settings) {
	vm.timeoutInSeconds.Set(s.TimeoutInSeconds)
}
