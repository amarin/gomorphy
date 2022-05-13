package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"git.media-tel.ru/railgo/logging"
	"git.media-tel.ru/railgo/logging/zap"
	"github.com/chzyer/readline"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
	"github.com/amarin/gomorphy/pkg/opencorpora"
)

const (
	cmdTag  = "tag"
	cmdSet  = "set"
	cmdExit = "exit"
	cmdNode = "node"
)

var (
	ErrTag  = errors.New(cmdTag)
	ErrSet  = errors.New(cmdSet)
	ErrNode = errors.New(cmdNode)
)

func outputTagsIndex(idx *index.Index, logger logging.Logger) {
	logger.Debugf("Tags index:")
	for tagID, tag := range idx.Tags() {
		logger.Debugf("[%d] %s (%s)", tagID, tag.Name.String(), tag.Parent.String())
	}
}

func processSearch(idx *index.Index, logger logging.Logger, line string) {
	var (
		forms dag.Node
		err   error
	)
	started := time.Now()
	if forms, err = idx.FetchString(line); err != nil {
		logger.Infof("< %v, eta %v", err, time.Since(started))
		return
	}

	tagSets := forms.TagSets()
	if len(tagSets) == 0 {
		logger.Infof("< empty, eta %v", time.Since(started))
		return
	} else {
		for formIdx, tagSet := range forms.TagSets() {
			logger.Infof("< %02d: %v, eta %v", formIdx, tagSet, time.Since(started))
		}
		return
	}
}

func processTag(idx *index.Index, logger logging.Logger, items ...string) error {
	const (
		subCmdCount = "count"
	)
	if len(items) != 2 {
		return fmt.Errorf("%w: expected 2 items", ErrTag)
	}

	var (
		tagId int
		err   error
		arg   = items[1]
	)
	tags := idx.Tags()
	switch arg {
	case subCmdCount:
		logger.Infof("%v.%v: %v", cmdTag, subCmdCount, tags.Len())
		return nil
	default:
		if tagId, err = strconv.Atoi(arg); err != nil {
			for intTagId, tag := range tags {
				if strings.ToLower(string(tag.Name)) == strings.ToLower(arg) {
					logger.Infof("%s[%v] %v(%v)", cmdTag, intTagId, tag.Name, tag.Parent)
					return nil
				}
			}

			return fmt.Errorf("%w: expected int id: `%v`", ErrTag, arg)
		}

		if tagId >= tags.Len() {
			return fmt.Errorf("%w: no such tag id: `%v`", ErrTag, arg)
		}

		tag := tags[tagId]
		logger.Infof("%s[%v] %v(%v)", cmdTag, arg, tag.Name, tag.Parent)
		return nil
	}
}

func processSet(idx *index.Index, logger logging.Logger, items ...string) error {
	const (
		subCmdCount  = "count"
		subCmdList   = "list"
		subCmdTables = "tables"
		subCmdTable  = "table"
	)
	if len(items) < 2 {
		return fmt.Errorf("%w: at least 2 items required", ErrSet)
	}

	var subCmd = items[1]

	sets := idx.TagSetIndex()

	switch subCmd {
	case subCmdTables:
		logger.Infof("%v.%v: total %v", cmdSet, subCmdTables, sets.Len())
		tablesStr := make([]string, sets.Len())
		for tableIdx := 0; tableIdx < sets.Len(); tableIdx++ {
			table := sets[tableIdx]
			tablesStr[tableIdx] = fmt.Sprintf("%02d(%d)", tableIdx, table.Len())
		}
		logger.Infof("%v.%v: %v", cmdSet, subCmdTables, strings.Join(tablesStr, ", "))

		return nil
	case subCmdTable:
		if len(items) != 3 {
			return fmt.Errorf("%w.%v: 3 items required", ErrSet, subCmdTable)
		}

		tableID, err := strconv.Atoi(items[2])
		if err != nil {
			return fmt.Errorf("%w.%v: `%v`: int ID required", ErrSet, subCmdTable, items[2])
		}
		if tableID >= sets.Len() {
			return fmt.Errorf("%w.%v: `%v`: no such set table", ErrSet, subCmdTable, tableID)
		}
		table := sets[tableID]
		setsStrings := make([]string, table.Len())
		for setIdx, set := range table {
			setsStrings[setIdx] = set.String()
		}

		logger.Infof("%v.%v: %v: %v", cmdSet, subCmdTables, tableID, strings.Join(setsStrings, ", "))
		return nil

	default:
		return nil
	}
}

func processNode(idx *index.Index, logger logging.Logger, items ...string) error {
	const (
		subCmdCount = "count"
		subCmdInfo  = "info"
	)
	if len(items) < 2 {
		return fmt.Errorf("%w: at least 2 items required", ErrSet)
	}

	var subCmd = items[1]

	switch subCmd {
	case subCmdCount:
		logger.Infof("%v.%v: total %v", cmdNode, subCmdCount, idx.NodesCount())

		return nil
	default:
		if len(items) != 2 {
			return fmt.Errorf("%w.%v: 2 items required", ErrNode, subCmdInfo)
		}
		strNodeId := items[1]
		nodeID, err := strconv.Atoi(strNodeId)
		if err != nil {
			return fmt.Errorf("%w.%v: `%v`: int ID required", ErrNode, subCmdInfo, strNodeId)
		}
		if nodeID >= idx.NodesCount() {
			return fmt.Errorf("%w.%v: `%v`: no such node", ErrNode, subCmdInfo, strNodeId)
		}
		var item dag.Node

		if item, err = idx.Get(dag.ID(nodeID)); err != nil {
			return fmt.Errorf("%w.%v: `%v`: get node: %v", ErrNode, subCmdInfo, strNodeId, err)
		}

		word := item.Word()
		ts := item.TagSets()
		setsStrings := make([]string, len(ts))
		for tsIdx, tagSet := range ts {
			tagSetStrings := make([]string, len(tagSet))
			for tagIdx, tag := range tagSet {
				tagSetStrings[tagIdx] = string(tag.Name)
			}
			setsStrings[tsIdx] = "TS" + strconv.Itoa(tsIdx) + "(" + strings.Join(tagSetStrings, ",") + ")"
		}

		logger.Infof("node %v: %v: %v", strNodeId, word, strings.Join(setsStrings, ", "))

		return nil
	}
}

func processInput(idx *index.Index, logger logging.Logger, line string) {
	var err error

	items := strings.Split(line, " ")

	switch items[0] {
	case cmdTag:
		err = processTag(idx, logger, items...)
	case cmdSet:
		err = processSet(idx, logger, items...)
	case cmdNode:
		err = processNode(idx, logger, items...)
	case cmdExit:
		logger.Infof("exiting")
		os.Exit(0)
	default:
		processSearch(idx, logger, line)
	}

	if err != nil {
		logger.Error(err.Error())
		err = nil
	}
}

func main() {
	var (
		err    error
		logger logging.Logger
		idx    *index.Index
		rl     *readline.Instance
		line   string
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

	if logger.IsEnabledForLevel(logging.LevelDebug) {
		outputTagsIndex(idx, logger)
	}

	logger.Debugf("TagSetIndex: %v", idx.TagSetIndex().String())

	for {
		if line, err = rl.Readline(); err != nil { // io.EOF
			break
		}

		processInput(idx, logger, line)
	}

	os.Exit(0)
}
