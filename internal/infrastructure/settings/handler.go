package settings

import (
	"encoding/json"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

const (
	storageV1Key = "settings"
	storageV2Key = "settingsV2"
)

type handler struct {
	bus   event.Bus
	prefs fyne.Preferences
	mu    sync.Mutex
}

func FyneSettingsHandler(bus event.Bus, prefs fyne.Preferences) {
	h := &handler{bus: bus, prefs: prefs}

	bus.Subscribe().
		On(event.Is(settings.LoadTriggeredType), h.handleLoad).
		On(event.Is(settings.WriteTriggeredType), h.handleWrite).
		ListenWithWorkers(1)
}

func (h *handler) handleWrite(evt event.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	pl := evt.Payload().(settings.WriteTriggered)
	handleErr := func(err error) {
		h.bus.Publish(evt.NewFollowup(settings.WriteFailed{
			Name: pl.Name,
			Err:  err,
		}))
	}

	settingsDtos, err := fromJson[map[string]settingDTO](h.prefs.String(storageV2Key))
	if err != nil {
		handleErr(err)
	}

	newVal, err := newDto(pl.Name, pl.Value)
	if err != nil {
		handleErr(err)
	}

	settingsDtos[pl.Name] = newVal

	bytes, err := json.Marshal(settingsDtos)
	if err != nil {
		handleErr(err)
	}

	h.prefs.SetString(storageV2Key, string(bytes))

	h.bus.Publish(evt.NewFollowup(settings.WriteSucceeded{
		Name:  pl.Name,
		Value: pl.Value,
	}))
}

func (h *handler) handleLoad(evt event.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	handleErr := func(err error) {
		h.bus.Publish(evt.NewFollowup(settings.LoadFailed{
			Err: err,
		}))
	}

	settingsDtos, err := fromJson[map[string]settingDTO](h.prefs.String(storageV2Key))
	if err != nil {
		handleErr(err)
		return
	}

	values := make(map[string]any)
	registered := make(map[string]settings.SType)
	for name, dto := range settingsDtos {
		val, tp := dto.Read()
		if val == nil {
			handleErr(fmt.Errorf("invalid configuration type for the setting %s", name))
			return
		}
		values[name] = val
		registered[name] = tp
	}

	h.bus.Publish(evt.NewFollowup(settings.LoadSucceeded{
		Values:     values,
		Registered: registered,
	}))
}

func fromJson[T any](content string) (T, error) {
	var structType T
	err := json.Unmarshal([]byte(content), &structType)
	if err != nil {
		return structType, fmt.Errorf("fromJson: %w", err)
	}
	return structType, nil
}
