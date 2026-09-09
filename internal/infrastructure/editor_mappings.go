package infrastructure

import (
	"encoding/json"

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
	content := r.prefs.String(fynePrefKeyEditorMappings)
	if content == "" || content == "null" {
		return nil, nil
	}

	var mappings []editor.Mapping
	if err := json.Unmarshal([]byte(content), &mappings); err != nil {
		return nil, err
	}

	return mappings, nil
}

func (r *editorMappingsRepository) SaveAll(mappings []editor.Mapping) error {
	bytes, err := json.Marshal(mappings)
	if err != nil {
		return err
	}

	r.prefs.SetString(fynePrefKeyEditorMappings, string(bytes))
	return nil
}
