package explorer

type S3DirectoryID string

func (id S3DirectoryID) String() string {
	return string(id)
}

type S3Directory struct {
	ID                S3DirectoryID
	Name              string
	Parrent           *S3Directory
	SubDirectories    []*S3Directory
	SubDirectoriesIDs []S3DirectoryID
	Files             []*S3File
	IsLoaded          bool
}

const (
	rooDirName = "/"
	RootDirID  = S3DirectoryID("")
)

func NewS3Directory(name string, parent *S3Directory) *S3Directory {
	if parent == nil {
		parent = RootDir
	}
	d := &S3Directory{
		Name:           name,
		Parrent:        parent,
		SubDirectories: make([]*S3Directory, 0),
		Files:          make([]*S3File, 0),
		IsLoaded:       false,
	}
	d.ID = S3DirectoryID(d.Path())
	return d
}

var (
	RootDir = &S3Directory{
		Name:           rooDirName,
		Parrent:        nil,
		SubDirectories: make([]*S3Directory, 0),
		Files:          make([]*S3File, 0),
		IsLoaded:       false,
	}
)

func (d *S3Directory) AddSubdir(sd *S3Directory) {
	d.SubDirectories = append(d.SubDirectories, sd)
}

func (d *S3Directory) AddFile(f *S3File) {
	d.Files = append(d.Files, f)
}

func (d *S3Directory) Path() string {
	if d.Parrent == nil {
		return d.Name
	}
	if d.Parrent == RootDir {
		return d.Parrent.Path() + d.Name
	}
	return d.Parrent.Path() + "/" + d.Name
}

func (d *S3Directory) DisplayContent() string {
	var content = "-> " + d.Name + "\n"
	for _, sd := range d.SubDirectories {
		content += "\t-> " + sd.Name + "\n"
	}
	for _, f := range d.Files {
		content += "\t-  " + f.Name() + "\n"
	}
	return content
}
