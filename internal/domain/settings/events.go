package settings

import "github.com/thomas-marquis/it-happened/event"

const (
	LoadTriggeredType event.Type = "event.settings.load.triggered"
	LoadSucceededType event.Type = "event.settings.load.succeeded"
	LoadFailedType    event.Type = "event.settings.load.failed"
)

type LoadTriggered struct{}

func (LoadTriggered) EventType() event.Type {
	return LoadTriggeredType
}

type LoadSucceeded struct {
	Values     map[string]any
	Registered map[string]SType
}

func (LoadSucceeded) EventType() event.Type {
	return LoadSucceededType
}

type LoadFailed struct {
	Err error
}

func (LoadFailed) EventType() event.Type {
	return LoadFailedType
}

const (
	SaveSucceededType event.Type = "event.settings.save.succeeded"
	SaveFailedType    event.Type = "event.settings.save.failed"
)

type SaveSucceeded struct{}

func (SaveSucceeded) EventType() event.Type {
	return SaveSucceededType
}

type SaveFailed struct {
	Err    error
	Events []event.Event
}

func (SaveFailed) EventType() event.Type {
	return SaveFailedType
}

const (
	WriteTriggeredType event.Type = "event.settings.write.triggered"
	WriteSucceededType event.Type = "event.settings.write.succeeded"
	WriteFailedType    event.Type = "event.settings.write.failed"
)

type WriteTriggered struct {
	Name  string
	Value any
}

func (WriteTriggered) EventType() event.Type {
	return WriteTriggeredType
}

type WriteSucceeded struct {
	Name  string
	Value any
}

func (WriteSucceeded) EventType() event.Type {
	return WriteSucceededType
}

type WriteFailed struct {
	Name string
	Err  error
}

func (WriteFailed) EventType() event.Type {
	return WriteFailedType
}
