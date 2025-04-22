package viewmodel

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/settings"
)

const (
	settingsTimeout = 15 * time.Second
)

type SettingsViewModel struct {
	settingsRepo settings.Repository
	loading      binding.Bool
	errChan      chan error
}

func NewSettingsViewModel(settingsRepo settings.Repository) *SettingsViewModel {
	errChan := make(chan error)

	vm := &SettingsViewModel{
		settingsRepo: settingsRepo,
		loading:      binding.NewBool(),
		errChan:      errChan,
	}

	// Start error handler
	go func() {
		for err := range errChan {
			fmt.Printf("Error in SettingsViewModel: %v\n", err)
		}
	}()

	return vm
}

func (vm *SettingsViewModel) Save(s settings.Settings) error {
	if err := vm.settingsRepo.Save(s); err != nil {
		vm.errChan <- fmt.Errorf("error saving settings: %w", err)
		return err
	}

	return nil
}

func (vm *SettingsViewModel) Get() (settings.Settings, error) {
	s, err := vm.settingsRepo.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting settings: %w", err)
		return settings.Settings{}, err
	}

	return s, nil
}

func (vm *SettingsViewModel) Loading() binding.Bool {
	return vm.loading
}

func (vm *SettingsViewModel) StartLoading() {
	vm.loading.Set(true)
}

func (vm *SettingsViewModel) StopLoading() {
	vm.loading.Set(false)
} 