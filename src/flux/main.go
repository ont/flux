package main

import (
	"github.com/kataras/iris"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose     = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
	debugConfig = kingpin.Flag("debug-config", "Print debug ouput of parsed config").Short('d').Bool()
	config      = kingpin.Flag("config", "Config file").Short('c').Default("/etc/flux.conf").File()
)

func main() {
	kingpin.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	app := iris.New()

	grammar := NewGrammar(*config)

	PrintConfig(grammar, *debugConfig, *verbose)

	for _, route := range grammar.Routes {
		queue := NewConsumer(app, route)

		workers := NewWorkers(queue, route.Metrics)
		for _, worker := range workers {
			go worker.Start()
		}
	}

	_ = app.Run(
		iris.Addr(":8080"),
		iris.WithoutServerError(iris.ErrServerClosed),
		iris.WithOptimizations,
		iris.WithoutBanner,
		iris.WithoutVersionChecker,
	)

}
