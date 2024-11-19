package viewmodel

import "github.com/thomas-marquis/s3-box/internal/connection"

type AppState struct {
	SelectedConnection *connection.Connection
}

func NewAppState() AppState {
	return AppState{}
}
