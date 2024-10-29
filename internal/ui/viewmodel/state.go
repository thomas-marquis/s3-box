package viewmodel

import "go2s3/internal/connection"

type AppState struct {
	SelectedConnection *connection.Connection
}

func NewAppState() AppState {
	return AppState{}
}
