package explorer

type S3Directory struct {
	Name                string
	Parrent             *S3Directory
	SubDirectories      []*S3Directory // DEPRECATED
	Files               []*S3File
	IsLoaded            bool
	SubDirectoriesPaths []string
}

const (
	rooDirName = "/"
)

func NewDirectory(name string, parent *S3Directory) *S3Directory {
	if parent == nil {
		parent = RootDir
	}
	return &S3Directory{
		Name:                name,
		Parrent:             parent,
		SubDirectories:      make([]*S3Directory, 0),
		Files:               make([]*S3File, 0),
		IsLoaded:            false,
		SubDirectoriesPaths: make([]string, 0),
	}
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

func (d *S3Directory) CreateNewSubDirectory(name string) *S3Directory {
	return NewDirectory(name, d)
}

// func (d *S3Directory) AddSubdir(sd *S3Directory) {
// 	d.SubDirectories = append(d.SubDirectories, sd)
// }

func (d *S3Directory) AddSubdir(sdPath string) {
	d.SubDirectoriesPaths = append(d.SubDirectoriesPaths, sdPath)
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

func (d *S3Directory) IsRoot() bool {
	return d == RootDir
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

func (d *S3Directory) Unload() {
	d.IsLoaded = false
	d.Files = make([]*S3File, 0)
	d.SubDirectories = make([]*S3Directory, 0)
}
