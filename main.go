package main

import (
	"go2s3/internal/ui/app"
	"go2s3/internal/ui/app/navigation"

	"go.uber.org/zap"
)

func main() {
	logCfg := zap.NewDevelopmentConfig()
	logCfg.Level.SetLevel(zap.DebugLevel)
	logger, err := logCfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	a, err := app.New(logger, navigation.ExplorerRoute)
	if err != nil {
		panic(err)
	}

	err = a.Start()
	if err != nil {
		panic(err)
	}
}
