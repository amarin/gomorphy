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
	programDescription = "Download and unpack opencorpora.ru dictionary"
)

func initLogging(debug bool) {
	opts := []logging.Option{
		logging.WithFormat(logging.FormatText),
		logging.WithTarget(logging.StdErr),
	}
	if debug {
		opts = append(opts, logging.WithLevel(logging.LevelDebug))
	} else {
		opts = append(opts, logging.WithLevel(logging.LevelInfo))
	}
	if err := logging.Init(opts...); err != nil {
		fmt.Printf("logging: init: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	skipDownload := flag.Bool(
		"l",
		false,
		"use local file only, skip downloading.",
	)
	debugLogging := flag.Bool(
		"v",
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
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "%s - %s\n\n", path.Base(os.Args[0]), programDescription)
		flag.PrintDefaults()
		os.Exit(0)
	}

	initLogging(*debugLogging)

	// Compilation of unpacked dictionary into runtime format is added
	// at docs/todo.md stage 7.
	loader := opencorpora.NewLoader("")

	if err := loader.Update(*skipDownload); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
