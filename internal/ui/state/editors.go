package state

import (
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/texteditor"
)

type EditorsState struct {
	selector *editor.Selector
}

func newEditorsState() *EditorsState {
	return &EditorsState{
		selector: editor.NewSelector(),
	}
}

func (s *EditorsState) Selector() *editor.Selector {
	return s.selector
}

// GetDefaultFactory returns the default factory for the given file path.
// If no matching editor is found, it returns the text editor factory as default.
func (s *EditorsState) GetDefaultFactory(filePath string) editor.Factory {
	if s.selector == nil {
		return nil
	}
	factory, err := s.selector.Select(filePath)
	if err != nil {
		textFactory, err := s.selector.GetRegisteredEditorByName(texteditor.Name)
		if err == nil && textFactory != nil {
			return textFactory
		}
		registered := s.selector.RegisteredEditors()
		if len(registered) > 0 {
			return registered[0]
		}
		panic("no registered editors")
	}
	return factory
}
