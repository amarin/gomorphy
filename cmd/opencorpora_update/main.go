package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

const (
	programDescription = "Load and/or build compiled index from downloaded opencorpora.ru dictionary"
)

func main() {
	forceRecompile := flag.Bool(
		"f",
		true,
		"force rebuild index from previously downloaded data even if compiled index already present",
	)
	debugLogging := flag.Bool(
		"d",
		false,
		"switch on debug logging causes very noisy logging output",
	)
	usageOutput := flag.Bool(
		"h",
		false,
		"Output this usage screen",
	)

	flag.Parse()
	if *usageOutput {
		fmt.Fprintf(flag.CommandLine.Output(), "%s - %s\n\n", path.Base(os.Args[0]), programDescription)
		flag.PrintDefaults()
		os.Exit(0)
	}

	loggingOpts := make([]logging.Option, 0)
	if *debugLogging {
		loggingOpts = append(loggingOpts, logging.WithLevel(logging.LevelDebug))
	}
	if err := logging.Init(loggingOpts...); err != nil {
		fmt.Printf("logging: init: %v\n", err)
		os.Exit(1)
	}

	loader := opencorpora.NewLoader("")

	if err := loader.Update(*forceRecompile); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
