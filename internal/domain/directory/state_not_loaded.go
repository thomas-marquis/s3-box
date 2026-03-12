package directory

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

type notLoadedState struct {
	baseState
	status Status
}

var _ state = (*notLoadedState)(nil)

func newNotLoadedState(d *Directory, status Status) *notLoadedState {
	return &notLoadedState{baseState{d: d}, status}
}

func (s *notLoadedState) Type() StateType { return stateTypeNotLoaded }

func (s *notLoadedState) Load() (LoadEvent, error) {
	s.d.setState(newLoadingState(s.baseState))
	return NewLoadEvent(s.d), nil
}

func (s *notLoadedState) Notify(event.Event) error {
	return nil
}

func (s *notLoadedState) SubDirectories() ([]*Directory, error) {
	return nil, ErrNotLoaded
}

func (s *notLoadedState) Files() ([]*File, error) {
	return nil, ErrNotLoaded
}

func (s *notLoadedState) Status() Status {
	return s.status
}
