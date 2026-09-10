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
	global      *GlobalState
	editors     *EditorsState
}

func New() *State {
	st := &State{
		connections: newConnectionState(),
		settings:    newSettingsState(),
		tags:        newTagsState(),
		global:      newGlobalState(),
		editors:     newEditorsState(),
	}

	st.explorer = newExplorerState(st)

	return st
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

func (s *State) Global() *GlobalState {
	return s.global
}

func (s *State) Editors() *EditorsState {
	return s.editors
}
