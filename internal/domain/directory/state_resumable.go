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
	return nil
}

func (s *resumableState) Status() Status {
	return s.currentStatus
}

func (s *resumableState) Resume() (event.Event, error) {
	switch status := s.currentStatus.(type) {
	case RenamePendingStatus:
		return NewRenameResumeEvent(s.d, status.IsSourceDir, status.OtherDirPath), nil
	}
	return nil, nil
}
