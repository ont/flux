package main

import (
	"os"

	"flux/pkg/server"

	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose     = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
	debugConfig = kingpin.Flag("debug-config", "Print debug ouput of parsed config").Short('d').Bool()
	config      = kingpin.Flag("config", "Config file").Short('c').Default("/etc/flux.conf").File()
	test        = kingpin.Flag("test", "Test regexps from all metrics with this string").Short('t').String()
)

func main() {
	kingpin.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	app := iris.New()

	grammar := server.NewGrammar(*config)

	server.PrintConfig(grammar, *debugConfig, *verbose)

	if *test != "" {
		server.TestRegexps(grammar, *test)
		os.Exit(0)
	}

	rootConsumer := server.NewRootConsumer(app)

	for _, route := range grammar.Routes {
		consumer, queue := server.NewConsumer(app, route)
		rootConsumer.AddConsumer(route.Name, consumer)

		workers := server.NewWorkers(queue, route.Metrics)
		for _, worker := range workers {
			go worker.Start()
		}
	}

	_ = app.Run(
		iris.Addr(":8080"),
		iris.WithoutServerError(iris.ErrServerClosed),
		iris.WithOptimizations,
		iris.WithoutBanner,
	)

}
