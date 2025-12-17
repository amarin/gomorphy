package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"path"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/internal/app"
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

	app.InitLogging(*debugLogging)

	logger := logging.NewNamedLogger("opencorpora")
	logger.WithLevel(logging.LevelDebug)
	// init loader
	loader := opencorpora.NewLoader("")

	//// Start CPU profiling
	//go func() {
	//	http.ListenAndServe("localhost:8080", nil)
	//}()

	if err := loader.Update(*forceRecompile, *skipDownload); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
