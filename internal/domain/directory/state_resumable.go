package directory

import (
	"errors"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type resumableState struct {
	baseState
	currentStatus Status
}

var _ state = (*resumableState)(nil)

func newResumableState(previous baseState, status Status) *resumableState {
	return &resumableState{previous.Clone(), status}
}

func (s *resumableState) Type() StateType {
	return stateTypeResumable
}

func (s *resumableState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "this directory is already loaded")
}

func (s *resumableState) UploadFile(localPtah string, overwrite bool) (ContentUploadedEvent, error) {
	return ContentUploadedEvent{}, errors.New("you can't upload files to a resumable directory")
}

func (s *resumableState) Notify(evt event.Event) error {
	switch e := evt.(type) {

	case RenameSuccessEvent:
		s.d.name = e.NewName()
		s.d.path = s.d.parent.Path().NewSubPath(e.NewName())
		for _, file := range s.files {
			file.updateDirectoryPath(s.d.path)
		}
		for _, subDir := range s.subDirs {
			subDir.updatePath(s.d.path)
		}
		s.d.setState(newLoadedState(s.baseState.Clone(), s.subDirs, s.files))

	case RenameFailureEvent:
		var urErr UncompletedRename
		if errors.As(e.Error(), &urErr) {
			status := RenamePendingStatus{
				CurrentDirectory: s.d,
				IsSourceDir:      true,
				OtherDirPath:     s.d.ParentPath().NewSubPath(e.NewName()),
			}
			s.d.setState(newResumableState(s.baseState.Clone(), status))
		}
	}
	return nil
}

func (s *resumableState) Status() Status {
	return s.currentStatus
}

func (s *resumableState) Resume() (event.Event, error) {
	switch status := s.currentStatus.(type) {
	case RenamePendingStatus:
		var srcDir, dstDir *Directory

		parent := s.d.parent // no need to check parent nullity: renaming root dir is forbidden
		otherDir, err := parent.GetSubDirectoryByName(status.OtherDirPath.DirectoryName())
		if err != nil {
			return nil, err
		}

		if status.IsSourceDir {
			srcDir = s.d
			dstDir = otherDir
		} else {
			srcDir = otherDir
			dstDir = s.d
		}

		return NewRenameResumeEvent(srcDir, dstDir), nil
	}
	return nil, errors.New("nothing to resume")
}
