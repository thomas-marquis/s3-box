package directory

type UpdateType string

const (
	UpdateTypeCreated     UpdateType = "CREATED"
	UpdateTypeDeleted     UpdateType = "DELETED"
	UpdateTypeFileCreated UpdateType = "FILE_CREATED"
	UpdateTypeFileDeleted UpdateType = "FILE_DELETED"
	UpdateTypeNone        UpdateType = "NONE"
)

type Update struct {
	dir               *Directory
	updateType        UpdateType
	attachedFile      *File
	attachedDirectory Path
}

func NewUpdate(dir *Directory, eventType UpdateType) Update {
	return Update{dir, eventType, nil, ""}
}

// Object returns the directory that emitted the event.
func (u *Update) Object() *Directory {
	return u.dir
}

func (u *Update) Type() UpdateType {
	return u.updateType
}

// AttachedFile returns the file attached to the event, if any, nil otherwise.
func (u *Update) AttachedFile() *File {
	return u.attachedFile
}

// AttachedDirPath returns the directory path attached to the event, if any, empty Path otherwise.
func (u *Update) AttachedDirPath() Path {
	return u.attachedDirectory
}

func (u *Update) AttachFile(f *File) {
	if u.updateType != UpdateTypeFileCreated && u.updateType != UpdateTypeFileDeleted {
		panic("programming error: invalid event type for attaching a file")
	}
	u.attachedFile = f
}

func (u *Update) AttachDirectory(p Path) {
	if u.updateType != UpdateTypeCreated || u.updateType != UpdateTypeDeleted {
		panic("programming error: invalid event type for attaching a directory")
	}
	u.attachedDirectory = p
}
