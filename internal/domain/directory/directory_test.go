package directory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func TestDirectory(t *testing.T) {
	t.Run("should change directory states", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

		// When & Then
		// not loaded directory
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsOpened())

		// loading it
		evt, err := dir.Load()
		assert.NoError(t, err)
		assert.Equal(t, directory.LoadEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())
		assert.True(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsOpened())

		// loading ended sucssesffuly
		dir.SetLoaded(true)
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsOpened())

		// Then, open it
		dir.Open()
		assert.NoError(t, err)
		assert.True(t, dir.IsOpened())
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
	})

	t.Run("should change directory states with error", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

		// When
		dir.Load() //nolint:errcheck
		dir.SetLoaded(false)

		// Then
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsOpened())
	})
}
