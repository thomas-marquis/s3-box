package settings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

func TestSettings_load(t *testing.T) {
	t.Run("should load and merge incoming settings with existing ones", func(t *testing.T) {
		// Given
		remoteValues := map[string]any{
			"app.timeout":         10 * time.Second,
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
}
