package main

import (
	"github.com/thomas-marquis/s3-box/internal/ui/app"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"

	"go.uber.org/zap"
)

func main() {
	//mux := http.NewServeMux()
	//mux.HandleFunc("/debug/pprof/", pprof.Index)
	//mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	//mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	//mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	//mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	//mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	//mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	//mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	//mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	//
	//go func() {
	//	log.Println("Serveur pprof démarré sur http://localhost:6060")
	//	if err := http.ListenAndServe("localhost:6060", mux); err != nil {
	//		log.Printf("Erreur pprof: %v", err)
	//	}
	//}()
	//
	//time.Sleep(5 * time.Second)

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
