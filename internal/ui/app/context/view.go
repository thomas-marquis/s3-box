package appcontext

import (
	"fyne.io/fyne/v2"
)

type View func(AppContext) (*fyne.Container, error)
