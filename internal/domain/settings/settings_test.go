package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

func TestNewAPI(t *testing.T) {
	s := settings.NewSettingsV3()
	assert.IsType(t, settings.IdleState{}, s.State())

	// Test registration
	err := s.Register(settings.AString("app.theme", "dark"))
	require.NoError(t, err)

	// Test read
	assert.Equal(t, "dark", s.ReadString("app.theme"))

	// Test write
	err = s.Write("app.theme", "light")
	require.NoError(t, err)

	// Test load
	loadEvt, err := s.Load()
	require.NoError(t, err)
	assert.IsType(t, settings.LoadingState{}, s.State())

	// Test that write during load fails
	err = s.Write("app.theme", "blue")
	assert.ErrorIs(t, err, settings.ErrNotReady)

	// Simulate load succeeded
	_ = loadEvt
	succEvt := event.New(settings.LoadSucceeded{
		Values:     map[string]any{"app.theme": "loaded"},
		Registered: map[string]settings.SType{"app.theme": settings.StringType},
	})
	err = s.Notify(succEvt)
	require.NoError(t, err)
	assert.IsType(t, settings.IdleState{}, s.State())
	assert.Equal(t, "loaded", s.ReadString("app.theme"))

	// Test save
	err = s.Write("app.theme", "new")
	require.NoError(t, err)

	_, err = s.Save()
	require.NoError(t, err)
	assert.IsType(t, settings.SavingState{}, s.State())

	// Test that save during save fails
	_, err = s.Save()
	assert.ErrorIs(t, err, settings.ErrNotReady)

	// Test that load during save fails
	_, err = s.Load()
	assert.ErrorIs(t, err, settings.ErrNotReady)

	// Test that write during save is allowed
	err = s.Write("app.theme", "another")
	assert.NoError(t, err)

	// Simulate save succeeded
	saveSuccEvt := event.New(settings.SaveSucceeded{})
	err = s.Notify(saveSuccEvt)
	require.NoError(t, err)
	assert.IsType(t, settings.IdleState{}, s.State())

	// Test observer
	observeCalled := false
	var observedValue any
	unobserve := s.Observe("app.theme", func(value any) {
		observeCalled = true
		observedValue = value
	})
	defer unobserve()

	// Trigger observer
	writeEvt := event.New(settings.WriteSucceeded{Name: "app.theme", Value: "observed"})
	err = s.Notify(writeEvt)
	require.NoError(t, err)
	assert.True(t, observeCalled)
	assert.Equal(t, "observed", observedValue)
}
