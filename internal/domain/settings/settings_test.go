package settings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

func TestSettings_register(t *testing.T) {
	t.Run("should register a new string setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(settings.AString("app.theme", "dark"))

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.theme"))
		assert.True(t, s.IsExistsWithType("app.theme", settings.StringType))
		assert.Equal(t, "dark", s.ReadString("app.theme"))
	})

	t.Run("should register a new uint64 setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(settings.AUint64("app.maxRetries", 5))

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.maxRetries"))
		assert.True(t, s.IsExistsWithType("app.maxRetries", settings.Uint64Type))
		assert.Equal(t, uint64(5), s.ReadUint64("app.maxRetries"))
	})

	t.Run("should register a new duration setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(settings.ADuration("app.timeout", 30*time.Second))

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.timeout"))
		assert.True(t, s.IsExistsWithType("app.timeout", settings.DurationType))
		assert.Equal(t, 30*time.Second, s.ReadDuration("app.timeout"))
	})

	t.Run("should register multiple settings at once", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(
			settings.AString("app.theme", "dark"),
			settings.AUint64("app.maxRetries", 5),
			settings.ADuration("app.timeout", 30*time.Second),
		)

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.theme"))
		assert.True(t, s.IsExists("app.maxRetries"))
		assert.True(t, s.IsExists("app.timeout"))
		assert.Equal(t, "dark", s.ReadString("app.theme"))
		assert.Equal(t, uint64(5), s.ReadUint64("app.maxRetries"))
		assert.Equal(t, 30*time.Second, s.ReadDuration("app.timeout"))
	})

	t.Run("should fail when registering a duplicate setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When
		err := s.Register(settings.AString("app.theme", "light"))

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrAlreadyExists)
		assert.Contains(t, err.Error(), "app.theme")
	})
}

func TestSettings_write(t *testing.T) {
	t.Run("should add write event to pending events", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When
		err := s.Write("app.theme", "light")

		// Then
		assert.NoError(t, err)
		assert.False(t, s.IsReady())
	})

	t.Run("should fail when writing to unregistered setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Write("app.theme", "light")

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrUnregistered)
		assert.Contains(t, err.Error(), "app.theme")
	})

	t.Run("should fail when writing with wrong type", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When
		err := s.Write("app.theme", uint64(10))

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrUnregistered)
		assert.Contains(t, err.Error(), "app.theme")
	})
}

func TestSettings_isExists(t *testing.T) {
	t.Run("should return false for unregistered setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When & Then
		assert.False(t, s.IsExists("app.theme"))
	})

	t.Run("should return true for registered setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When & Then
		assert.True(t, s.IsExists("app.theme"))
	})
}

func TestSettings_isExistsWithType(t *testing.T) {
	t.Run("should return false for unregistered setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When & Then
		assert.False(t, s.IsExistsWithType("app.theme", settings.StringType))
	})

	t.Run("should return false for wrong type", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When & Then
		assert.False(t, s.IsExistsWithType("app.theme", settings.Uint64Type))
	})

	t.Run("should return true for correct type", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When & Then
		assert.True(t, s.IsExistsWithType("app.theme", settings.StringType))
	})
}

func TestSettings_readMethods(t *testing.T) {
	t.Run("should panic when reading unregistered string", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When & Then
		assert.Panics(t, func() {
			s.ReadString("app.theme")
		})
	})

	t.Run("should panic when reading unregistered uint64", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When & Then
		assert.Panics(t, func() {
			s.ReadUint64("app.maxRetries")
		})
	})

	t.Run("should panic when reading unregistered duration", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When & Then
		assert.Panics(t, func() {
			s.ReadDuration("app.timeout")
		})
	})

	t.Run("should return default value for string when not loaded yet", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When & Then
		assert.Equal(t, "dark", s.ReadString("app.theme"))
	})

	t.Run("should return default value for uint64 when not loaded yet", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AUint64("app.maxRetries", 5)))

		// When & Then
		assert.Equal(t, uint64(5), s.ReadUint64("app.maxRetries"))
	})

	t.Run("should return default value for duration when not loaded yet", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.ADuration("app.timeout", 30*time.Second)))

		// When & Then
		assert.Equal(t, 30*time.Second, s.ReadDuration("app.timeout"))
	})
}

func TestSettings_load(t *testing.T) {
	t.Run("should load and merge incoming settings with existing ones", func(t *testing.T) {
		// Given
		remoteValues := map[string]any{
			"app.timeout":         int64(10000000000),
			"app.colorTheme":      "dark",
			"app.maxFileSizeByte": uint64(1024),
			"app.other":           "value",
		}
		remoteRegistered := map[string]settings.SType{
			"app.timeout":         settings.DurationType,
			"app.colorTheme":      settings.StringType,
			"app.maxFileSizeByte": settings.Uint64Type,
			"app.other":           settings.StringType,
		}

		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(
			settings.AString("app.colorTheme", "white"),
			settings.AUint64("app.maxFileSizeByte", 20*1024),
			settings.AUint64("app.maxConcurency", 3),
			settings.ADuration("app.timeout", 30*time.Second)))

		// When
		evt, err := s.Load()
		assert.NoError(t, err)
		assert.False(t, s.IsReady())

		succEvt := evt.NewFollowup(settings.LoadSucceeded{
			Values:     remoteValues,
			Registered: remoteRegistered,
		})
		err2 := s.Notify(succEvt)

		// Then
		assert.NoError(t, err2)
		assert.True(t, s.IsReady())

		assert.Equal(t, 10*time.Second, s.ReadDuration("app.timeout"))
		assert.Equal(t, "dark", s.ReadString("app.colorTheme"))
		assert.Equal(t, uint64(1024), s.ReadUint64("app.maxFileSizeByte"))
		assert.Equal(t, uint64(3), s.ReadUint64("app.maxConcurency"))
		assert.False(t, s.IsExists("app.other"))
	})

	t.Run("should set isReady to false when loading", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		_, err := s.Load()

		// Then
		assert.NoError(t, err)
		assert.False(t, s.IsReady())
	})
}

func TestSettings_save(t *testing.T) {
	t.Run("should return success event when no pending events", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		evt := s.Save()

		// Then
		assert.NotNil(t, evt)
	})

	t.Run("should return carrier event when pending events exist", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))

		// When
		evt := s.Save()

		// Then
		assert.NotNil(t, evt)
		assert.False(t, s.IsReady())
	})

	t.Run("should clear pending events after save", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))

		// When
		s.Save()

		// Then
		// After save, isReady should be false and new save should work
		assert.False(t, s.IsReady())
		assert.NotNil(t, s.Save())
	})
}

func TestSettings_notify(t *testing.T) {
	t.Run("should handle LoadSucceeded and merge values", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(
			settings.AString("app.theme", "dark"),
			settings.ADuration("app.timeout", 30*time.Second),
		))

		// When
		loadEvt, _ := s.Load()
		succEvt := loadEvt.NewFollowup(settings.LoadSucceeded{
			Values: map[string]any{
				"app.theme":   "light",
				"app.timeout": int64(60000000000),
			},
			Registered: map[string]settings.SType{
				"app.theme":   settings.StringType,
				"app.timeout": settings.DurationType,
			},
		})
		err := s.Notify(succEvt)

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsReady())
		assert.Equal(t, "light", s.ReadString("app.theme"))
		assert.Equal(t, 60*time.Second, s.ReadDuration("app.timeout"))
	})

	t.Run("should handle SaveSucceeded and set isReady", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		saveEvt := event.New(settings.SaveSucceeded{})
		err := s.Notify(saveEvt)

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsReady())
	})

	t.Run("should handle WriteSucceeded and update value", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: "light"})
		err := s.Notify(writeEvt)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "light", s.ReadString("app.theme"))
	})
}

func TestSettings_observe(t *testing.T) {
	t.Run("should call observer when value changes", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		observeCalled := false
		var observedValue any
		unobserve := s.Observe("app.theme", func(value any) {
			observeCalled = true
			observedValue = value
		})
		defer unobserve()

		// When
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: "light"})
		s.Notify(writeEvt)

		// Then
		assert.True(t, observeCalled)
		assert.Equal(t, "light", observedValue)
	})

	t.Run("should not call observer for different setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(
			settings.AString("app.theme", "dark"),
			settings.AString("app.color", "red"),
		))

		observeCalled := false
		unobserve := s.Observe("app.theme", func(value any) {
			observeCalled = true
		})
		defer unobserve()

		// When
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.color", Value: "blue"})
		s.Notify(writeEvt)

		// Then
		assert.False(t, observeCalled)
	})

	t.Run("should stop calling observer after unobserve", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		observeCalled := false
		unobserve := s.Observe("app.theme", func(value any) {
			observeCalled = true
		})

		// When
		unobserve()
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: "light"})
		s.Notify(writeEvt)

		// Then
		assert.False(t, observeCalled)
	})

	t.Run("should allow multiple observers for same setting", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		observeCalled1 := false
		observeCalled2 := false
		unobserve1 := s.Observe("app.theme", func(value any) {
			observeCalled1 = true
		})
		defer unobserve1()
		unobserve2 := s.Observe("app.theme", func(value any) {
			observeCalled2 = true
		})
		defer unobserve2()

		// When
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: "light"})
		s.Notify(writeEvt)

		// Then
		assert.True(t, observeCalled1)
		assert.True(t, observeCalled2)
	})
}
