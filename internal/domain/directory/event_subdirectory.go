package directory

type SubDirectoryEvent interface {
	Event
	SubDirectory() Path
}

type subDirectoryDeletedEvent struct {
	baseEvent
	subDirectory Path
}

func newDirectoryDeletedEvent(parent *Directory, subDirectory Path) subDirectoryDeletedEvent {
	return subDirectoryDeletedEvent{baseEvent{parent}, subDirectory}
}

func (e subDirectoryDeletedEvent) Name() string {
	return SubDirectoryDeletedEventName
}

func (e subDirectoryDeletedEvent) SubDirectory() Path {
	return e.subDirectory
}
