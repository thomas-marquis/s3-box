package uiutils

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func ToFyneUris(uris []string) ([]fyne.URI, error) {
	var res []fyne.URI
	for _, uri := range uris {
		fUri, err := storage.ParseURI(uri)
		if err != nil {
			return nil, err
		}
		res = append(res, fUri)
	}
	return res, nil
}

func FromFyneUrisToPaths(uris []fyne.URI) []string {
	var res []string
	for _, uri := range uris {
		res = append(res, uri.Path())
	}
	return res
}

func UrisToPaths(uris []fyne.URI) []string {
	var res []string
	for _, uri := range uris {
		res = append(res, uri.Path())
	}
	return res
}

func GetCommonParentPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	res := paths[0]
	for _, p := range paths[1:] {
		if len(filepath.SplitList(p)) < len(filepath.SplitList(res)) {
			res = p
		}
	}

	res = strings.TrimSuffix(res, string(filepath.Separator))

	return filepath.Dir(res)

}
