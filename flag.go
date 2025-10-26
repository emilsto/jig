package main

import (
	"os"
	"flag"
	"log"
)

func handleCommandLine() bool {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			config, err := getOrCreateConfig("config.toml")
			if err != nil {
				log.Fatal(err)
			}
			if err := handleInitJigrc(config); err != nil {
				log.Fatal(err)
			}
			return true
		}
	}
	return false
}

func parseFlags() (help, epics, oneshot bool) {
	flagHelp := flag.Bool("h", false, "Show help message")
	flagEpics := flag.Bool("e", false, "Fetch epics from project")
	flagOneshot := flag.Bool("o", false, "Run once and exit (oneshot mode)")
	flag.Parse()
	return *flagHelp, *flagEpics, *flagOneshot
}


