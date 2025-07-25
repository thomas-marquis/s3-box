package directory

const (
	CreatedEventName             = "event.directory.created"
	SubDirectoryDeletedEventName = "event.directory.deleted"
	FileCreatedEventName         = "event.file.created"
	FileDeletedEventName         = "event.file.deleted"
)

type Event interface {
	Name() string
	Directory() *Directory
}

type baseEvent struct {
	parent *Directory
}

func (e baseEvent) Directory() *Directory {
	return e.parent
}

type createdEvent struct {
	baseEvent
}

func newDirectoryCreatedEvent(directory *Directory) createdEvent {
	return createdEvent{baseEvent{directory}}
}

func (e createdEvent) Name() string {
	return CreatedEventName
}
