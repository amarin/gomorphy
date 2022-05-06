package main

import (
	"fmt"
	"os"
	"time"

	"git.media-tel.ru/railgo/logging"
	"git.media-tel.ru/railgo/logging/zap"
	"github.com/chzyer/readline"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
	"github.com/amarin/gomorphy/pkg/opencorpora"
)

func main() {
	var (
		line   string
		err    error
		logger logging.Logger
		idx    *index.Index
		rl     *readline.Instance
		forms  dag.Node
	)

	loggingConfig := *logging.CurrentConfig()
	loggingConfig.Level = logging.LevelDebug

	if err = logging.Init(loggingConfig, new(zap.Backend)); err != nil {
		fmt.Printf("logging: init: %v\n", err)
		os.Exit(1)
	}

	logger = logging.NewNamedLogger("opencorpora")
	logger.WithLevel(logging.LevelDebug)

	started := time.Now()
	loader := opencorpora.NewLoader("")
	if idx, err = loader.LoadIndex(); err != nil {
		logger.Error("load index: %v", err)
		os.Exit(1)
	}

	logger.Debug("loaded in ", time.Since(started))
	logger.Debugf("indexed %d words %d nodes", idx.WordsCount(), idx.NodesCount())

	if rl, err = readline.New("> "); err != nil {
		logger.Error("readline: %v", err)
		os.Exit(1)
	}

	defer func() {
		if err = rl.Close(); err != nil {
			logger.Warnf("readline: close: %v", err)
		}
	}()

	for {
		if line, err = rl.Readline(); err != nil { // io.EOF
			break
		}

		started = time.Now()
		if forms, err = idx.FetchString(line); err != nil {
			logger.Infof("< %v, eta %v", err, time.Since(started))
			continue
		}

		tagSets := forms.TagSets()
		if len(tagSets) == 0 {
			logger.Infof("< empty, eta %v", time.Since(started))
			continue
		} else {
			for formIdx, tagSet := range forms.TagSets() {
				logger.Infof("< %02d: %v, eta %v", formIdx, tagSet, time.Since(started))
			}
			continue
		}
	}

	os.Exit(0)
}
