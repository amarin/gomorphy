package app

import (
	"fmt"
	"os"

	"github.com/amarin/logging"
)

// InitLogging inits logging facility.
// Takes boolean trigger to switch on debug logging when true, else use default info level.
// Returns silently when logging subsystem ready or exits application with status code 1 in case logging init failed.
func InitLogging(debugLogging bool) {
	loggingOpts := []logging.Option{
		logging.WithLevel(logging.LevelInfo),
		logging.WithFormat(logging.FormatText),
		logging.WithTarget(logging.StdErr),
	}
	if debugLogging {
		loggingOpts = append(loggingOpts, logging.WithLevel(logging.LevelDebug))
	} else {
		loggingOpts = append(loggingOpts, logging.WithLevel(logging.LevelInfo))
	}

	if err := logging.Init(loggingOpts...); err != nil {
		fmt.Printf("logging: init: %v\n", err)
		os.Exit(1)
	}
}
