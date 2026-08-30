package state

import (
	"log"
	"os"
)

var (
	logger = log.New(os.Stdout, "[state] ", log.LstdFlags|log.Lshortfile)
)

type State struct {
	connections *ConnectionState
	explorer    *ExplorerState
	settings    *SettingsState
	tags        *TagsState
	ui          *UIState
}

func New() *State {
	return &State{
		connections: newConnectionState(),
		explorer:    newExplorerState(),
		settings:    newSettingsState(),
		tags:        newTagsState(),
		ui:          newUiState(),
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

func (s *State) UI() *UIState {
	return s.ui
}
