package directory

import (
	"errors"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type errorState struct {
	baseState
	currentStatus Status
}

var _ state = (*errorState)(nil)

func newErrorState(previous baseState, status Status) *errorState {
	return &errorState{previous.Clone(), status}
}

func (s *errorState) Type() StateType {
	return stateTypeError
}

func (s *errorState) Load() (LoadEvent, error) {
	// reload
	s.d.setState(newLoadingState(s.baseState))
	return NewLoadEvent(s.d), nil
}

func (s *errorState) UploadFile(localPtah string, overwrite bool) (ContentUploadedEvent, error) {
	return ContentUploadedEvent{}, errors.New("you can't upload files to a resumable directory")
}

func (s *errorState) Notify(evt event.Event) error {
	switch e := evt.(type) {

	case RenameSuccessEvent:
		s.d.name = e.NewName()
		s.d.path = s.d.parent.Path().NewSubPath(e.NewName())
		for _, subDir := range s.subDirs {
			subDir.updatePath(s.d.path)
		}
		s.d.setState(newLoadedState(s.baseState, s.subDirs, s.files))

	case RenameFailureEvent:
		if errors.Is(e.Error(), &UncompletedRename{}) {
			status := RenameFailedStatus{
				CurrentDirectory: s.d,
				IsSourceDir:      true,
				OtherDirPath:     s.d.ParentPath().NewSubPath(e.NewName()),
			}
			s.d.setState(newErrorState(s.baseState, status))
		}
	}
	return nil
}

func (s *errorState) Status() Status {
	return s.currentStatus
}

func (s *errorState) Recover(choice RecoveryChoice) (event.Event, error) {
	switch status := s.currentStatus.(type) {
	case RenameFailedStatus:
		var srcDir, dstDir *Directory

		if status.IsSourceDir {
			srcDir = s.d
		} else {
			dstDir = s.d
		}

		parent := s.d.parent // no need to check parent nullity: renaming root dir is forbidden
		otherDir, err := parent.GetSubDirectoryByName(status.OtherDirPath.DirectoryName())
		if err != nil {
			if errors.Is(err, ErrNotFound) && choice == RecoveryChoiceRenameAbort {
				// meh...
				s.d.setState(newLoadingState(s.baseState))
				return NewRenameRecoverEvent(srcDir, dstDir, choice), nil
			}
			return nil, err
		}

		if status.IsSourceDir {
			dstDir = otherDir
		} else {
			srcDir = otherDir
		}

		s.d.setState(newLoadingState(s.baseState))
		otherDir.setState(newLoadingState(baseState{d: otherDir}))
		return NewRenameRecoverEvent(srcDir, dstDir, choice), nil
	}
	return nil, errors.New("nothing to recover")
}
