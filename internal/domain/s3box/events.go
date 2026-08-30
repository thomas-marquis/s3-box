package s3box

import "github.com/thomas-marquis/it-happened/event"

const (
	UserValidationAskedType    event.Type = "event.s3box.user.validation.asked"
	UserValidationAcceptedType event.Type = "event.s3box.user.validation.accepted"
	UserValidationRefusedType  event.Type = "event.s3box.user.validation.refused"
)

type UserValidationAsked struct {
	Reason  event.Event
	Message string
}

func (UserValidationAsked) EventType() event.Type {
	return UserValidationAskedType
}

type UserValidationAccepted struct {
	Reason event.Event
}

func (UserValidationAccepted) EventType() event.Type {
	return UserValidationAcceptedType
}

type UserValidationRefused struct {
	Reason event.Event
}

func (UserValidationRefused) EventType() event.Type {
	return UserValidationRefusedType
}
