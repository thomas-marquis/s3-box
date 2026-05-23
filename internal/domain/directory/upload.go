package directory

type UploadMode int

const (
	UploadModeSkip UploadMode = iota
	UploadModeReplace
	UploadModeDuplicate
	UploadModeDefault = UploadModeSkip
)

type FsItem struct {
	Name     string
	AbsPath  string
	IsDir    bool
	Children []FsItem
}

type UploadedItemPreview struct {
	Name                     string
	IsDir, IsNew, IsReplaced bool
	Children                 []UploadedItemPreview
}
