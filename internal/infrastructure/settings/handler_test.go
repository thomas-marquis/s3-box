package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/eventest"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	infra "github.com/thomas-marquis/s3-box/internal/infrastructure/settings"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_fyne "github.com/thomas-marquis/s3-box/mocks/fyne"
	"go.uber.org/mock/gomock"
)

func TestFyneSettingsHandler_write(t *testing.T) {
	t.Run("should write the new settings when it exists", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		existingPrefs := `{
			"timeout": {
				"name": "timeout",
				"nsValue": 10000000
			},
			"maxConcurrency": {
				"name": "maxConcurrency",
				"u64Value": 3
			},
			"lang": {
				"name": "lang",
				"strValue": "en"
			}
		}`
		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return(existingPrefs).Times(1)

		newPrefs := `{
			"timeout": {
				"name": "timeout",
				"nsValue": 10000000
			},
			"maxConcurrency": {
				"name": "maxConcurrency",
				"u64Value": 3
			},
			"lang": {
				"name": "lang",
				"strValue": "fr"
			}
		}`
		mockPrefs.EXPECT().
			SetString(gomock.Eq("settingsV2"), testutil.JsonEqMatcher(t, newPrefs)).
			Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(eventest.PayloadEq(settings.WriteSucceeded{
				Name:  "lang",
				Value: "fr",
			})).
			Do(func(evt event.Event) {
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.WriteTriggered{
			Name:  "lang",
			Value: "fr",
		})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should handle write when preferences are empty", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		// Empty preferences string
		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return("").Times(1)
		mockPrefs.EXPECT().
			SetString(gomock.Eq("settingsV2"), gomock.Any()).
			Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(eventest.PayloadEq(settings.WriteSucceeded{
				Name:  "lang",
				Value: "fr",
			})).
			Do(func(evt event.Event) {
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.WriteTriggered{
			Name:  "lang",
			Value: "fr",
		})

		// Then
		testutil.AssertEventually(t, done)
	})
}

func TestFyneSettingsHandler_load(t *testing.T) {
	t.Run("should load existing settings", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		existingPrefs := `{
			"timeout": {
				"name": "timeout",
				"nsValue": 10000000
			},
			"maxConcurrency": {
				"name": "maxConcurrency",
				"u64Value": 3
			},
			"lang": {
				"name": "lang",
				"strValue": "en"
			}
		}`
		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return(existingPrefs).Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(eventest.PayloadEq(settings.LoadSucceeded{
				Values: map[string]any{
					"timeout":        int64(10000000),
					"maxConcurrency": uint64(3),
					"lang":           "en",
				},
				Registered: map[string]settings.SType{
					"timeout":        settings.DurationType,
					"maxConcurrency": settings.Uint64Type,
					"lang":           settings.StringType,
				},
			})).
			Do(func(evt event.Event) {
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.LoadTriggered{})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should return empty maps when no settings exist", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return("{}").Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(eventest.PayloadEq(settings.LoadSucceeded{
				Values:     map[string]any{},
				Registered: map[string]settings.SType{},
			})).
			Do(func(evt event.Event) {
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.LoadTriggered{})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should handle empty string from prefs and return empty maps", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		// Empty string (no settings stored yet)
		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return("").Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(eventest.PayloadEq(settings.LoadSucceeded{
				Values:     map[string]any{},
				Registered: map[string]settings.SType{},
			})).
			Do(func(evt event.Event) {
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.LoadTriggered{})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should fail when preferences cannot be read", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return("invalid json").Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl := evt.Payload().(settings.LoadFailed)
				assert.ErrorContains(t, pl.Err, "fromJson")
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.LoadTriggered{})

		// Then
		testutil.AssertEventually(t, done)
	})

	t.Run("should fail when DTO has invalid configuration type", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		existingPrefs := `{
			"invalidSetting": {
				"name": "invalidSetting"
			}
		}`
		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return(existingPrefs).Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(gomock.Any()).
			Do(func(evt event.Event) {
				pl := evt.Payload().(settings.LoadFailed)
				assert.ErrorContains(t, pl.Err, "invalid configuration type")
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.LoadTriggered{})

		// Then
		testutil.AssertEventually(t, done)
	})
}
