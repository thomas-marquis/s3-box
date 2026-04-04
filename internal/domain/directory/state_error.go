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

func (s *errorState) Load() (event.Event, error) {
	// reload
	s.d.setState(newLoadingState(s.baseState))
	return event.New(LoadTriggered{Directory: s.d}), nil
}

func (s *errorState) UploadFile(localPtah string, overwrite bool) (event.Event, error) {
	return event.Event{}, errors.New("you can't upload files to a resumable directory")
}

func (s *errorState) Notify(evt event.Event) error {
	switch pl := evt.Payload.(type) {

	case RenameSucceeded:
		s.d.name = pl.NewName
		s.d.path = s.d.parent.Path().NewSubPath(pl.NewName)
		for _, subDir := range s.subDirs {
			subDir.updatePath(s.d.path)
		}
		s.d.setState(newLoadedState(s.baseState, s.subDirs, s.files))

	case RenameFailed:
		if errors.Is(pl.Err, &UncompletedRename{}) {
			status := RenameFailedStatus{
				CurrentDirectory: s.d,
				IsSourceDir:      true,
				OtherDirPath:     s.d.ParentPath().NewSubPath(pl.NewName),
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
				return event.New(RenameRecoveryTriggered{
					Directory: srcDir,
					DstDir:    dstDir,
					Choice:    choice,
				}), nil
			}
			return event.Event{}, err
		}

		if status.IsSourceDir {
			dstDir = otherDir
		} else {
			srcDir = otherDir
		}

		s.d.setState(newLoadingState(s.baseState))
		otherDir.setState(newLoadingState(baseState{d: otherDir}))
		return event.New(RenameRecoveryTriggered{
			Directory: srcDir,
			DstDir:    dstDir,
			Choice:    choice,
		}), nil
	}
	return event.Event{}, errors.New("nothing to recover")
}
