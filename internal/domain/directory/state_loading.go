package directory

import (
	"errors"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type loadingState struct {
	baseState
}

var _ state = (*loadingState)(nil)

func newLoadingState(previous baseState) *loadingState {
	return &loadingState{previous.Clone()}
}

func (s *loadingState) Type() StateType {
	return stateTypeLoading
}

func (s *loadingState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "loading is still in progress")
}

func (s *loadingState) Notify(evt event.Event) error {
	switch e := evt.(type) {
	case LoadSuccessEvent:
		s.d.setState(newLoadedState(s.Clone(), e.SubDirectories(), e.Files()))

	case LoadFailureEvent:
		var urErr UncompletedRename
		if errors.As(e.Error(), &urErr) {
			isSrc := s.d.Path() == urErr.SourceDirPath
			var other Path
			if isSrc {
				other = urErr.DestinationDirPath
			} else {
				other = urErr.SourceDirPath
			}
			status := RenamePendingStatus{
				CurrentDirectory: s.d,
				IsSourceDir:      isSrc,
				OtherDirPath:     other,
			}

			s.d.setState(newResumableState(s.baseState.Clone(), status))
			return nil
		}

		s.d.setState(newNotLoadedState(s.d, ErrorStatus{Err: e.Error()}))
	}

	return nil
}

func (s *loadingState) Open() {}

func (s *loadingState) Close() {}

func (s *loadingState) SubDirectories() ([]*Directory, error) {
	return nil, ErrNotLoaded
}

func (s *loadingState) Files() ([]*File, error) {
	return nil, ErrNotLoaded
}
