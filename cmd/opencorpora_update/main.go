package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

const programDescription = "Download, unpack and compile opencorpora.ru dictionary"

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
	skipCompile := flag.Bool(
		"skip-compile",
		false,
		"download/unpack only, do not compile the runtime dictionary.",
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
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "%s - %s\n\n", path.Base(os.Args[0]), programDescription) //nolint:forbidigo
		flag.PrintDefaults()

		os.Exit(0)
	}

	initLogging(*debugLogging)

	started := time.Now()

	loader := opencorpora.NewLoader("")

	if err := loader.Update(*skipDownload); err != nil {
		os.Exit(1)
	}

	if *skipCompile {
		logDone(started, "compile skipped")
	} else if !loader.IsCompiledExists() {
		os.Exit(1)
	} else {
		logDone(started, "dictionary ready at "+loader.CompiledFilePath())
	}

	os.Exit(0)
}

// logDone prints total wall time and peak memory of the whole cycle.
func logDone(started time.Time, msg string) {
	var mem runtime.MemStats

	runtime.ReadMemStats(&mem)

	_, _ = fmt.Fprintf(os.Stderr, "done in %s, peak RSS heap %d MiB: %s\n", //nolint:forbidigo
		time.Since(started).Round(time.Millisecond), mem.Sys>>20, msg)
}
