package directory

import (
	"fmt"
	"strings"
)

type Status interface {
	Title() string
	Message() string
}

type RenameFailedStatus struct {
	CurrentDirectory *Directory
	IsSourceDir      bool
	OtherDirPath     Path
}

func (s RenameFailedStatus) Title() string {
	return "Rename pending"
}

func (s RenameFailedStatus) Message() string {
	msg := strings.Builder{}
	fmt.Fprintf(&msg, "A rename operation is pending for this directory ") // nolint:errcheck
	srcPath := s.CurrentDirectory.Path()
	dstPath := s.OtherDirPath
	if !s.IsSourceDir {
		srcPath, dstPath = dstPath, srcPath
	}
	fmt.Fprintf(&msg, "from '%s' to '%s'", srcPath.DirectoryName(), dstPath.DirectoryName()) // nolint:errcheck
	return msg.String()

}

type ErrorStatus struct {
	Err error
}

func (s ErrorStatus) Title() string {
	return "Error"
}

func (s ErrorStatus) Message() string {
	return s.Err.Error()
}
