package explorer

type Directory struct {
	Name           string
	Parrent        *Directory
	SubDirectories []*Directory
	Files          []*RemoteFile
	IsLoaded       bool
}

const (
	rooDirName = "/"
)

func NewDirectory(name string, parent *Directory) *Directory {
	if parent == nil {
		parent = RootDir
	}
	return &Directory{
		Name:           name,
		Parrent:        parent,
		SubDirectories: make([]*Directory, 0),
		Files:          make([]*RemoteFile, 0),
		IsLoaded:       false,
	}
}

var (
	RootDir = &Directory{
		Name:           rooDirName,
		Parrent:        nil,
		SubDirectories: make([]*Directory, 0),
		Files:          make([]*RemoteFile, 0),
		IsLoaded:       false,
	}
)

func (d *Directory) AddSubdir(sd *Directory) {
	d.SubDirectories = append(d.SubDirectories, sd)
}

func (d *Directory) AddFile(f *RemoteFile) {
	d.Files = append(d.Files, f)
}

func (d *Directory) Path() string {
	if d.Parrent == nil {
		return d.Name
	}
	if d.Parrent == RootDir {
		return d.Parrent.Path() + d.Name
	}
	return d.Parrent.Path() + "/" + d.Name
}

func (d *Directory) IsRoot() bool {
	return d == RootDir
}

func (d *Directory) DisplayContent() string {
	var content = "-> " + d.Name + "\n"
	for _, sd := range d.SubDirectories {
		content += "\t-> " + sd.Name + "\n"
	}
	for _, f := range d.Files {
		content += "\t-  " + f.Name() + "\n"
	}
	return content
}

func (d *Directory) Unload() {
	d.IsLoaded = false
	d.Files = make([]*RemoteFile, 0)
	d.SubDirectories = make([]*Directory, 0)
}
