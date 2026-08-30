package state

import "github.com/thomas-marquis/s3-box/internal/domain/s3box"

const (
	maxPendingUserValidations = 30
)

type UIState struct {
	pendingUserValidation chan s3box.UserValidationAsked
}

func newUiState() *UIState {
	return &UIState{
		pendingUserValidation: make(chan s3box.UserValidationAsked, maxPendingUserValidations),
	}
}

func (s *UIState) PendingUserValidation() chan s3box.UserValidationAsked {
	return s.pendingUserValidation
}
