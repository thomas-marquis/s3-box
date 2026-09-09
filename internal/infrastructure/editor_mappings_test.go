package infrastructure_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	mocks_fyne "github.com/thomas-marquis/s3-box/mocks/fyne"
	"go.uber.org/mock/gomock"
)

func TestEditorMappingsRepository_GetAll(t *testing.T) {
	t.Run("should return empty list when no mappings stored", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)

		mockPrefs.EXPECT().
			String(gomock.Eq("settings.editors.mappings")).
			Return("").
			Times(1)

		repo := infrastructure.NewEditorMappingsRepository(mockPrefs)

		// When
		mappings, err := repo.GetAll()

		// Then
		assert.NoError(t, err)
		assert.Empty(t, mappings)
	})

	t.Run("should return empty list when stored value is null", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)

		mockPrefs.EXPECT().
			String(gomock.Eq("settings.editors.mappings")).
			Return("null").
			Times(1)

		repo := infrastructure.NewEditorMappingsRepository(mockPrefs)

		// When
		mappings, err := repo.GetAll()

		// Then
		assert.NoError(t, err)
		assert.Empty(t, mappings)
	})

	t.Run("should return mappings when stored", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)

		mappingsJSON := `[{"regexp":"\\.csv$","name":"csv"},{"regexp":"\\.txt$","name":"text"}]`

		mockPrefs.EXPECT().
			String(gomock.Eq("settings.editors.mappings")).
			Return(mappingsJSON).
			Times(1)

		repo := infrastructure.NewEditorMappingsRepository(mockPrefs)

		// When
		mappings, err := repo.GetAll()

		// Then
		assert.NoError(t, err)
		assert.Len(t, mappings, 2)
		assert.Equal(t, "\\.csv$", mappings[0].RegexpPattern)
		assert.Equal(t, "csv", mappings[0].EditorName)
		assert.Equal(t, "\\.txt$", mappings[1].RegexpPattern)
		assert.Equal(t, "text", mappings[1].EditorName)
	})

	t.Run("should return error when json is invalid", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)

		mockPrefs.EXPECT().
			String(gomock.Eq("settings.editors.mappings")).
			Return("invalid json").
			Times(1)

		repo := infrastructure.NewEditorMappingsRepository(mockPrefs)

		// When
		mappings, err := repo.GetAll()

		// Then
		assert.Error(t, err)
		assert.Nil(t, mappings)
	})
}

func TestEditorMappingsRepository_SaveAll(t *testing.T) {
	t.Run("should save mappings successfully", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)

		mappings := []editor.Mapping{
			{RegexpPattern: "\\.csv$", EditorName: "csv"},
			{RegexpPattern: "\\.txt$", EditorName: "text"},
		}

		mockPrefs.EXPECT().
			SetString(gomock.Eq("settings.editors.mappings"), gomock.Eq(`[{"regexp":"\\.csv$","name":"csv"},{"regexp":"\\.txt$","name":"text"}]`)).
			Times(1)

		repo := infrastructure.NewEditorMappingsRepository(mockPrefs)

		// When
		err := repo.SaveAll(mappings)

		// Then
		assert.NoError(t, err)
	})

	t.Run("should save empty list", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)

		mappings := []editor.Mapping{}

		mockPrefs.EXPECT().
			SetString(gomock.Eq("settings.editors.mappings"), gomock.Eq(`[]`)).
			Times(1)

		repo := infrastructure.NewEditorMappingsRepository(mockPrefs)

		// When
		err := repo.SaveAll(mappings)

		// Then
		assert.NoError(t, err)
	})
}
