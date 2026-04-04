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

func (s *loadingState) Load() (event.Event, error) {
	return event.Event{}, NewError(s.d, "loading is still in progress")
}

func (s *loadingState) Notify(evt event.Event) error {
	switch pl := evt.Payload.(type) {
	case LoadSucceeded:
		s.d.setState(newLoadedState(s.baseState, pl.SubDirectories, pl.Files))

	case LoadFailed:
		var urErr UncompletedRename
		if errors.As(pl.Err, &urErr) {
			isSrc := s.d.Path() == urErr.SourceDirPath
			var otherPath Path
			if isSrc {
				otherPath = urErr.DestinationDirPath
			} else {
				otherPath = urErr.SourceDirPath
			}

			otherDir, err := s.d.parent.GetSubDirectoryByName(otherPath.DirectoryName())
			if err == nil && !otherDir.HasError() {
				otherDir.setState(newErrorState(baseState{d: otherDir},
					RenameFailedStatus{
						CurrentDirectory: otherDir,
						IsSourceDir:      !isSrc,
						OtherDirPath:     s.d.Path(),
					}))
			} else if err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}

			s.d.setState(newErrorState(s.baseState, RenameFailedStatus{
				CurrentDirectory: s.d,
				IsSourceDir:      isSrc,
				OtherDirPath:     otherPath,
			}))
			return nil
		}

		s.d.setState(newNotLoadedState(s.d, ErrorStatus{Err: pl.Err}))

	case RenameSucceeded:
		s.d.name = pl.NewName
		s.d.path = s.d.parent.Path().NewSubPath(pl.NewName)
		for _, subDir := range s.subDirs {
			subDir.updatePath(s.d.path)
		}
		s.d.setState(newLoadedState(s.Clone(), s.subDirs, s.files))

	case RenameFailed:
		var urErr UncompletedRename
		if errors.As(pl.Err, &urErr) {
			status := RenameFailedStatus{
				CurrentDirectory: s.d,
				IsSourceDir:      true,
				OtherDirPath:     s.d.ParentPath().NewSubPath(pl.NewName),
			}
			s.d.setState(newErrorState(s.baseState, status))
		}
		s.d.setState(newNotLoadedState(s.d, ErrorStatus{Err: pl.Err}))

	case UserValidationRefused:
		s.d.setState(newLoadedState(s.baseState, s.subDirs, s.files))
	}

	return nil
}

func (s *loadingState) Open() {}

func (s *loadingState) Close() {}
