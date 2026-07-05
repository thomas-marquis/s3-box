package state

import (
	"log"
	"os"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
)

var (
	logger = log.New(os.Stdout, "[state] ", log.LstdFlags|log.Lshortfile)
)

type ConnectionsState struct{}

type State struct {
	connections *ConnectionsState
	explorer    *ExplorerState
	settings    *SettingsState
}

func New() *State {
	return &State{
		connections: &ConnectionsState{},
		explorer: &ExplorerState{
			fileTree: binding.NewTree[node.Node](func(n1 node.Node, n2 node.Node) bool {
				return n1.ID() == n2.ID()
			}),
		},
		settings: &SettingsState{
			aggregate: &settings.SettingsV3{},
		},
	}
}

func (s *State) Explorer() *ExplorerState {
	return s.explorer
}

func (s *State) Connections() *ConnectionsState {
	return s.connections
}

func (s *State) Settings() *SettingsState {
	return s.settings
}
