package main

import (
	"os"
	"time"

	"github.com/chzyer/readline"
	"github.com/sirupsen/logrus"

	"github.com/amarin/gomorphy/internal/text"
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

	started := time.Now()

	grammemesIndex, err := loader.LoadGrammemes()
	if err != nil {
		logger.Error(err)
		logger.Error("Use opencorpora_update to download and compile it")
		os.Exit(1)
	}

	logger.Debug("loaded in ", time.Since(started))

	started = time.Now()
	wordsIndex, err := loader.LoadLemmata(grammemesIndex)
	if err != nil {
		os.Exit(1)
	}

	logger.Debug("loaded in ", time.Since(started))

	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		started = time.Now()
		forms := wordsIndex.SearchForms(text.RussianText(line))
		for idx, grammemesList := range forms {
			logger.Info(idx, ". ", line, " ", grammemesList.String())
		}
		logger.Debug("searched in ", time.Since(started))
	}

	os.Exit(0)
}
