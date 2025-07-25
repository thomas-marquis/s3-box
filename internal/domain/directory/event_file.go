package directory

type FileEvent interface {
	Event
	File() *File
}

type fileCreatedEvent struct {
	baseEvent
	file *File
}

func newFileCreatedEvent(parent *Directory, file *File) fileCreatedEvent {
	return fileCreatedEvent{baseEvent{parent}, file}
}

func (e fileCreatedEvent) Name() string {
	return FileCreatedEventName
}

func (e fileCreatedEvent) File() *File {
	return e.file
}

type fileDeletedEvent struct {
	baseEvent
	file *File
}

func newFileDeletedEvent(parent *Directory, file *File) fileDeletedEvent {
	return fileDeletedEvent{baseEvent{parent}, file}
}

func (e fileDeletedEvent) Name() string {
	return FileDeletedEventName
}

func (e fileDeletedEvent) File() *File {
	return e.file
}
