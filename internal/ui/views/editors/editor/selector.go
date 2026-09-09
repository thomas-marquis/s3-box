package editor

import (
	"errors"
	"regexp"
	"slices"

	"github.com/thomas-marquis/s3-box/internal/u"
)

var (
	ErrEditorNotRegistered = errors.New("editor not registered")
	ErrNoMatchingEditor    = errors.New("no matching editor found")
)

type filePattern string

func (p filePattern) String() string {
	return string(p)
}

func (p filePattern) Matches(filePath string) bool {
	return u.SkipV(regexp.MatchString(p.String(), filePath))
}

// Mapping represents a mapping between a file pattern and an editor name.
type Mapping struct {
	RegexpPattern string `json:"regexp"`
	EditorName    string `json:"name"`
}

// CompareMappings returns true if two mappings are equal.
func CompareMappings(a, b Mapping) bool {
	return a.RegexpPattern == b.RegexpPattern && a.EditorName == b.EditorName
}

// MappingRepository defines the interface for storing and retrieving editor mappings.
type MappingRepository interface {
	GetAll() ([]Mapping, error)
	SaveAll(mappings []Mapping) error
}

// Selector allows managing multiple editors and their associated file patterns.
// It can be used to determine which editor to use for a given file based on its path.
type Selector struct {
	registeredEditors map[string]Factory
	mapping           map[filePattern]string
}

// NewSelector creates a new Selector instance.
func NewSelector() *Selector {
	return &Selector{
		registeredEditors: make(map[string]Factory),
		mapping:           make(map[filePattern]string),
	}
}

// RegisterEditor registers a new editor with the given name, display label, and factory function.
func (s *Selector) RegisterEditor(factory Factory) *Selector {
	s.registeredEditors[factory.Name()] = factory
	return s
}

func (s *Selector) RegisteredEditors() []Factory {
	editors := make([]Factory, 0, len(s.registeredEditors))
	for _, editor := range s.registeredEditors {
		editors = append(editors, editor)
	}
	return editors
}

// RegisterMapping registers a mapping between a file pattern and an editor name.
// Multiple mappings can be registered for the same editor, allowing it to handle different file types.
// An editor must be registered before it can be mapped to a file pattern, an error will be returned otherwise.
func (s *Selector) RegisterMapping(editorName, regexpPattern string) error {
	if _, exists := s.registeredEditors[editorName]; !exists {
		return ErrEditorNotRegistered
	}
	s.mapping[filePattern(regexpPattern)] = editorName
	return nil
}

// Mappings returns the list of registered mappings sorted in ascending priority order (from the more specific to the less specific).
func (s *Selector) Mappings() []Mapping {
	mappings := make([]Mapping, 0, len(s.mapping))
	for pattern, editorName := range s.mapping {
		mappings = append(mappings, Mapping{
			RegexpPattern: pattern.String(),
			EditorName:    editorName,
		})
	}
	slices.SortFunc(mappings, func(a, b Mapping) int {
		return len(a.RegexpPattern) - len(b.RegexpPattern)
	})
	return mappings
}

// Select returns the factory function for the editor that matches the given file path.
// If no editor matches the file path, an ErrNoMatchingEditor error is returned.
func (s *Selector) Select(filePath string) (Factory, error) {
	for _, m := range s.Mappings() {
		if filePattern(m.RegexpPattern).Matches(filePath) {
			if factory, exists := s.registeredEditors[m.EditorName]; exists {
				return factory, nil
			}
		}
	}
	return nil, ErrNoMatchingEditor
}
