package state

import "github.com/thomas-marquis/s3-box/internal/domain/s3box"

const (
	maxPendingUserValidations = 30
)

type GlobalState struct {
	pendingUserValidation chan s3box.UserValidationAsked
}

func newUiState() *GlobalState {
	return &GlobalState{
		pendingUserValidation: make(chan s3box.UserValidationAsked, maxPendingUserValidations),
	}
}

func (s *GlobalState) PendingUserValidation() chan s3box.UserValidationAsked {
	return s.pendingUserValidation
}
