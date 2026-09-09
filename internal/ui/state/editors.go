package state

import "github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"

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
