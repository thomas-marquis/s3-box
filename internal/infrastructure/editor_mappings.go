package infrastructure

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	fynePrefKeyEditorMappings = "settings.editors.mappings"
)

type editorMappingsRepository struct {
	prefs fyne.Preferences
}

func NewEditorMappingsRepository(prefs fyne.Preferences) editor.MappingRepository {
	return &editorMappingsRepository{
		prefs: prefs,
	}
}

func (r *editorMappingsRepository) GetAll() ([]editor.Mapping, error) {
	return nil, nil
}

func (r *editorMappingsRepository) SaveAll(mappings []editor.Mapping) error {
	return nil
}
