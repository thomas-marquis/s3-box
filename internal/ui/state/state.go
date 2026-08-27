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
}

func New() *State {
	return &State{
		connections: newConnectionState(),
		explorer:    newExplorerState(),
		settings:    newSettingsState(),
		tags:        newTagsState(),
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
