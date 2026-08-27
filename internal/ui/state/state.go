package state

import (
	"log"
	"os"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
)

var (
	logger = log.New(os.Stdout, "[state] ", log.LstdFlags|log.Lshortfile)
)

type State struct {
	connections *ConnectionState
	explorer    *ExplorerState
	settings    *SettingsState
	tags        *TagsState
}

func New() *State {
	return &State{
		connections: newConnectionState(),
		explorer: &ExplorerState{
			fileTree: binding.NewTree[node.Node](func(n1 node.Node, n2 node.Node) bool {
				return n1.ID() == n2.ID()
			}),
		},
		settings: newSettingsState(),
		tags:     newTagsState(),
	}
}

func (s *State) Explorer() *ExplorerState {
	return s.explorer
}

func (s *State) Connection() *ConnectionState {
	return s.connections
}

func (s *State) Settings() *SettingsState {
	return s.settings
}

func (s *State) Tags() *TagsState {
	return s.tags
}
