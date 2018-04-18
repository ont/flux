package main

import (
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
)

func PrintConfig(grammar *Grammar, debugConfig, verbose bool) {
	if debugConfig {
		printGrammar(grammar, verbose)
		os.Exit(0)
	} else {
		total := 0
		for _, route := range grammar.Routes {
			total += len(route.Metrics)
		}
		log.Infof("Loaded config with %d routes and %d metrics total", len(grammar.Routes), total)
	}
}

func printGrammar(grammar *Grammar, verbose bool) {
	if verbose {
		cs := spew.NewDefaultConfig()
		cs.MaxDepth = 5
		cs.Dump(grammar)
		return
	}

	for _, route := range grammar.Routes {
		fmt.Printf("== Route \"%s\"\n", route.Name)
		for _, metric := range route.Metrics {
			fmt.Printf("    --> Metric \"%s\"\n", metric.Name)
			fmt.Printf("            regexp = \"%s\"\n", metric.Get("regexp"))

			if metric.eventName != "" {
				fmt.Printf("            event = \"%s\"\n", metric.Get("event"))
			}

			if metric.script != nil {
				fmt.Printf("            .. has script ..\n")
			}
		}
	}
}

func testRegexps(grammar *Grammar, data string) {
	for _, route := range grammar.Routes {
		fmt.Printf("== Route \"%s\"\n", route.Name)
		for _, metric := range route.Metrics {
			fmt.Printf("    --> Metric \"%s\"\n", metric.Name)

			matches := metric.re.FindStringSubmatch(data)
			if len(matches) > 0 {
				fmt.Printf("    [MATCH] \"%s\" --> %s\n", metric.Name, metric.re.String())
				for n, match := range matches {
					name := metric.re.SubexpNames()[n]
					if name == "" {
						name = fmt.Sprintf("group_%d", n)
					}
					fmt.Printf("           %10s --> %s\n", name, match)
				}
			} else {
				fmt.Printf("    [     ] \"%s\" --> %s\n", metric.Name, metric.re.String())
			}
		}
	}
}
