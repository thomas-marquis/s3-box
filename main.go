package main

import (
	"github.com/thomas-marquis/s3-box/internal/ui/app"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"

	"go.uber.org/zap"
)

func main() {
	logCfg := zap.NewDevelopmentConfig()
	logCfg.Level.SetLevel(zap.DebugLevel)
	logger, err := logCfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck

	a, err := app.New(logger, navigation.ExplorerRoute)
	if err != nil {
		panic(err)
	}

	err = a.Start()
	if err != nil {
		panic(err)
	}
}
