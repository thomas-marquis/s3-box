package directory

import (
	"fmt"
	"strings"
)

type Status interface {
	Title() string
	Message() string
}

type RenamePendingStatus struct {
	CurrentDirectory *Directory
	IsSourceDir      bool
	OtherDirPath     Path
}

func (s RenamePendingStatus) Title() string {
	return "Rename pending"
}

func (s RenamePendingStatus) Message() string {
	msg := strings.Builder{}
	msg.WriteString("A rename operation is pending for this directory ")
	srcPath := s.CurrentDirectory.Path()
	dstPath := s.OtherDirPath
	if !s.IsSourceDir {
		srcPath, dstPath = dstPath, srcPath
	}
	msg.WriteString(fmt.Sprintf("from '%s' to '%s'", srcPath, dstPath))
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
