package main

import (
	"flag"
	"fmt"
	"os"

	"git.media-tel.ru/railgo/logging"
	"git.media-tel.ru/railgo/logging/zap"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

func main() {
	var forceRecompile bool
	flag.BoolVar(
		&forceRecompile,
		"f",
		true,
		"set to yes if required rebuild index from downloaded data",
	)

	loggingConfig := *logging.CurrentConfig()
	loggingConfig.Level = logging.LevelDebug

	if err := logging.Init(loggingConfig, new(zap.Backend)); err != nil {
		fmt.Printf("logging: init: %v\n", err)
		os.Exit(1)
	}

	logger := logging.NewNamedLogger("opencorpora")
	logger.WithLevel(logging.LevelDebug)
	// init loader
	loader := opencorpora.NewLoader("")

	if err := loader.Update(forceRecompile); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
