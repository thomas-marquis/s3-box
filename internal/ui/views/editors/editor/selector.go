package editor

import (
	"regexp"

	"github.com/thomas-marquis/s3-box/internal/u"
)

// Selector is a struct that holds a regex pattern and a factory for creating editors.
type Selector struct {
	Pattern string
	Name    string
	Factory Factory `json:"-"`
}

// CanOpen checks if the given filename matches the selector's pattern
func (s *Selector) CanOpen(filename string) bool {
	return u.SkipV(regexp.MatchString(s.Pattern, filename))
}

func CompareSelector(a, b *Selector) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && a.Pattern == b.Pattern
}
