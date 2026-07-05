package settings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

func TestSettings_Register(t *testing.T) {
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
		err := s.Register(settings.ADuration("app.timeout", 30))

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.timeout"))
		assert.True(t, s.IsExistsWithType("app.timeout", settings.DurationType))
		assert.Equal(t, 30*time.Nanosecond, s.ReadDuration("app.timeout"))
	})

	t.Run("should register multiple settings at once", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(
			settings.AString("app.theme", "dark"),
			settings.AUint64("app.maxRetries", 5),
			settings.ADuration("app.timeout", 30*time.Nanosecond),
		)

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.theme"))
		assert.True(t, s.IsExists("app.maxRetries"))
		assert.True(t, s.IsExists("app.timeout"))
		assert.Equal(t, "dark", s.ReadString("app.theme"))
		assert.Equal(t, uint64(5), s.ReadUint64("app.maxRetries"))
		assert.Equal(t, 30*time.Nanosecond, s.ReadDuration("app.timeout"))
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

	t.Run("should fail when registering with empty name", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(settings.AString("", "dark"))

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrInvalidType)
	})

	t.Run("should fail when registering with whitespace-only name", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		err := s.Register(settings.AString("   ", "dark"))

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrInvalidType)
	})

	t.Run("should fail when registering during load", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		_, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When
		err = s.Register(settings.AString("app.theme", "dark"))

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrNotReady)
	})

	t.Run("should allow registration after load completes", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		loadEvt, err := s.Load()
		require.NoError(t, err)

		succEvt := loadEvt.NewFollowup(settings.LoadSucceeded{
			Values:     map[string]any{},
			Registered: map[string]settings.SType{},
		})
		err = s.Notify(succEvt)
		require.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())

		// When
		err = s.Register(settings.AString("app.theme", "dark"))

		// Then
		assert.NoError(t, err)
		assert.True(t, s.IsExists("app.theme"))
		assert.IsType(t, settings.IdleState{}, s.State())
	})
}

func TestSettings_Write(t *testing.T) {
	t.Run("should add write event to pending events", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When
		err := s.Write("app.theme", "light")

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())
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
		assert.ErrorIs(t, err, settings.ErrInvalidType)
		assert.Contains(t, err.Error(), "app.theme")
	})

	t.Run("should fail when writing during load", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		_, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When
		err = s.Write("app.theme", "light")

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrNotReady)
	})

	t.Run("should allow writing during save", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "pending"))
		_, err := s.Save()
		require.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())

		// When
		err = s.Write("app.theme", "light")

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())
	})
}

func TestSettings_Read(t *testing.T) {
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
		require.NoError(t, s.Register(settings.ADuration("app.timeout", 30)))

		// When & Then
		assert.Equal(t, 30*time.Nanosecond, s.ReadDuration("app.timeout"))
	})

	t.Run("should allow read during load", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		_, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When & Then
		assert.Equal(t, "dark", s.ReadString("app.theme"))
	})

	t.Run("should allow read during save", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		_, err := s.Save()
		require.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())

		// When & Then
		assert.Equal(t, "dark", s.ReadString("app.theme"))
	})
}

func TestSettings_Load(t *testing.T) {
	t.Run("should return LoadTriggered event and transition to LoadingState", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		assert.IsType(t, settings.IdleState{}, s.State())

		// When
		evt, err := s.Load()

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, evt)
		assert.IsType(t, settings.LoadingState{}, s.State())
		assert.Equal(t, settings.LoadTriggered{}, evt.Payload())
	})

	t.Run("should fail when loading during load", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		_, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When
		_, err = s.Load()

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrNotReady)
	})

	t.Run("should fail when loading during save", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		_, err := s.Save()
		require.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())

		// When
		_, err = s.Load()

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrNotReady)
	})

	t.Run("should transition to IdleState on LoadSucceeded", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(
			settings.AString("app.colorTheme", "white"),
			settings.AUint64("app.maxFileSizeByte", 20*1024),
			settings.AUint64("app.maxConcurrency", 3),
			settings.ADuration("app.timeout", 30*time.Nanosecond),
		))
		loadEvt, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When
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
		succEvt := loadEvt.NewFollowup(settings.LoadSucceeded{
			Values:     remoteValues,
			Registered: remoteRegistered,
		})
		err = s.Notify(succEvt)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())
		assert.Equal(t, 10*time.Second, s.ReadDuration("app.timeout"))
		assert.Equal(t, "dark", s.ReadString("app.colorTheme"))
		assert.Equal(t, uint64(1024), s.ReadUint64("app.maxFileSizeByte"))
		assert.Equal(t, uint64(3), s.ReadUint64("app.maxConcurrency"))
		assert.False(t, s.IsExists("app.other"))
	})

	t.Run("should transition to IdleState on LoadFailed", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		loadEvt, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When
		failEvt := loadEvt.NewFollowup(settings.LoadFailed{
			Err: assert.AnError,
		})
		err = s.Notify(failEvt)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())
	})
}

func TestSettings_Save(t *testing.T) {
	t.Run("should return SaveSucceeded when no pending events", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		assert.IsType(t, settings.IdleState{}, s.State())

		// When
		evt, err := s.Save()

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, evt)
		assert.IsType(t, settings.IdleState{}, s.State())
	})

	t.Run("should return carrier event and transition to SavingState when pending events exist", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		assert.IsType(t, settings.IdleState{}, s.State())

		// When
		evt, err := s.Save()

		// Then
		assert.NoError(t, err)
		assert.NotNil(t, evt)
		assert.IsType(t, settings.SavingState{}, s.State())
	})

	t.Run("should fail when saving during load", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		_, err := s.Load()
		require.NoError(t, err)
		assert.IsType(t, settings.LoadingState{}, s.State())

		// When
		_, err = s.Save()

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrNotReady)
	})

	t.Run("should fail when saving during save", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		_, err := s.Save()
		require.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())

		// When
		_, err = s.Save()

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrNotReady)
	})

	t.Run("should transition to IdleState on SaveSucceeded", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		_, err := s.Save()
		require.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())

		// When
		saveSuccEvt := event.New(settings.SaveSucceeded{})
		err = s.Notify(saveSuccEvt)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())
	})

	t.Run("should transition to IdleState and restore pending on SaveFailed", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))
		require.NoError(t, s.Write("app.theme", "light"))
		_, err := s.Save()
		require.NoError(t, err)
		assert.IsType(t, settings.SavingState{}, s.State())

		// When
		saveFailEvt := event.New(settings.SaveFailed{
			Err:    assert.AnError,
			Events: []event.Event{event.New(settings.WriteTriggered{Name: "app.theme", Value: "light"})},
		})
		err = s.Notify(saveFailEvt)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())
	})
}

func TestSettings_Notify(t *testing.T) {
	t.Run("should handle LoadSucceeded and merge values", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(
			settings.AString("app.theme", "dark"),
			settings.ADuration("app.timeout", 30*time.Nanosecond),
		))
		loadEvt, _ := s.Load()

		// When
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
		assert.IsType(t, settings.IdleState{}, s.State())
		assert.Equal(t, "light", s.ReadString("app.theme"))
		assert.Equal(t, 60*time.Second, s.ReadDuration("app.timeout"))
	})

	t.Run("should handle SaveSucceeded as a no-op", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		assert.IsType(t, settings.IdleState{}, s.State())

		// When
		saveEvt := event.New(settings.SaveSucceeded{})
		err := s.Notify(saveEvt)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, settings.IdleState{}, s.State())
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

	t.Run("should handle WriteSucceeded for unregistered setting as no-op", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()

		// When
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: "light"})
		err := s.Notify(writeEvt)

		// Then
		assert.NoError(t, err)
	})

	t.Run("should handle WriteSucceeded with wrong type", func(t *testing.T) {
		// Given
		s := settings.NewSettingsV3()
		require.NoError(t, s.Register(settings.AString("app.theme", "dark")))

		// When
		writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: uint64(10)})
		err := s.Notify(writeEvt)

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, settings.ErrInvalidType)
	})
}

func TestSettings_Observe(t *testing.T) {
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
		assert.NoError(t, s.Notify(writeEvt))

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
		assert.NoError(t, s.Notify(writeEvt))

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
		assert.NoError(t, s.Notify(writeEvt))

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
		assert.NoError(t, s.Notify(writeEvt))

		// Then
		assert.True(t, observeCalled1)
		assert.True(t, observeCalled2)
	})
}

func TestSettings_IsExists(t *testing.T) {
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

func TestSettings_IsExistsWithType(t *testing.T) {
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
