package directory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func TestFile_Rename(t *testing.T) {
	t.Run("should rename file and emit event", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		parentDir, err := directory.New(connID, "parent", directory.RootPath)
		require.NoError(t, err)

		file, err := directory.NewFile("oldname.txt", parentDir.Path())
		require.NoError(t, err)

		// When
		evt, err := file.Rename("newname.txt")

		// Then
		require.NoError(t, err)
		assert.Equal(t, directory.FileRenamedEventType, evt.Type())
		assert.Equal(t, directory.FileName("newname.txt"), file.Name())
		assert.Equal(t, directory.FileName("oldname.txt"), evt.OldName())
	})

	t.Run("should return error when new name is invalid", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		parentDir, err := directory.New(connID, "parent", directory.RootPath)
		require.NoError(t, err)

		file, err := directory.NewFile("oldname.txt", parentDir.Path())
		require.NoError(t, err)

		// When & Then - empty name
		_, err = file.Rename("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file name is empty")

		// When & Then - name with slash
		_, err = file.Rename("new/name.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file name is not valid")

		// When & Then - name is just slash
		_, err = file.Rename("/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file name is not valid")
	})

	t.Run("should update file in directory on rename success event", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		parentDir, err := directory.New(connID, "parent", directory.RootPath)
		require.NoError(t, err)

		file, err := directory.NewFile("oldname.txt", parentDir.Path())
		require.NoError(t, err)

		loadEvt := directory.NewLoadSuccessEvent(parentDir, nil, []*directory.File{file})
		_, err = parentDir.Load()
		require.NoError(t, err)
		require.NoError(t, parentDir.Notify(loadEvt))

		oldName := file.Name()
		_, err = file.Rename("newname.txt")
		require.NoError(t, err)

		// When
		successEvt := directory.NewFileRenamedSuccessEvent(parentDir, file, oldName)
		err = parentDir.Notify(successEvt)

		// Then
		require.NoError(t, err)
		files, _ := parentDir.Files()
		require.Len(t, files, 1)
		assert.Equal(t, directory.FileName("newname.txt"), files[0].Name())
	})
}
