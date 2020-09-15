package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "15:04:05.000000",
		DisableSorting:   true,
	})
	logger.SetLevel(logrus.DebugLevel)
	// init loader
	loader := opencorpora.NewLoader(logger, "")

	if err := loader.Update(false); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
