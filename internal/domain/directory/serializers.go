package directory

import "encoding/json"

var (
	_ json.Marshaler = (*File)(nil)
	_ json.Marshaler = (*Directory)(nil)
)

func (f *File) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":          f.Name().String(),
		"size":          f.SizeBytes(),
		"lastModified":  f.LastModified(),
		"directoryPath": f.DirectoryPath(),
		"fullPath":      f.FullPath(),
		"parentDirName": f.Parent().Name(),
	})
}

func (d *Directory) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":          d.Name(),
		"path":          d.Path(),
		"parentDirName": d.parent.Name(),
		"isOpened":      d.IsOpened(),
		"isRoot":        d.IsRoot(),
		"nbSubDirs":     len(d.SubDirectories()),
		"nbFiles":       len(d.Files()),
		"connectionID":  d.connectionID.String(),
	})
}
