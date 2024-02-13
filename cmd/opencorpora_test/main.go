package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/amarin/logging"
	"github.com/chzyer/readline"

	"github.com/amarin/gomorphy/internal/app"
	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
	"github.com/amarin/gomorphy/pkg/opencorpora"
)

const (
	cmdTag         = "tag"
	cmdTagShort    = "t"
	cmdSet         = "set"
	cmdSetShort    = "s"
	cmdVar         = "var"
	cmdVarShort    = "v"
	cmdExit        = "exit"
	cmdExitShort   = "e"
	cmdNode        = "node"
	cmdNodeShort   = "n"
	cmdReload      = "reload"
	cmdReloadShort = "r"
)

var (
	idx *index.Index

	ErrTag  = errors.New(cmdTag)
	ErrSet  = errors.New(cmdSet)
	ErrVar  = errors.New(cmdVar)
	ErrNode = errors.New(cmdNode)
)

func processSearch(logger logging.Logger, line string) {
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

func processTag(logger logging.Logger, items ...string) error {
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

func processSet(logger logging.Logger, items ...string) error {
	const (
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
		logger.Infof("%v.%v: total %v", cmdSet, subCmdTables, len(sets))
		tablesStr := make([]string, len(sets))
		for tableIdx := 0; tableIdx < len(sets); tableIdx++ {
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
		if tableID >= len(sets) {
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
		logger.Infof("%v.%v: looking tag set %v", cmdSet, subCmd, subCmd)
		tsNumber, err := strconv.ParseInt(subCmd, 0, 0)
		if err != nil {
			return fmt.Errorf("%w.%v: expected int or hex set number", ErrSet, subCmd)
		}

		logger.Infof("%v.%v(0x%0X): load tag set", cmdSet, tsNumber, tsNumber)

		tsIdx := idx.TagSetIndex()
		ts, ok := tsIdx.Get(index.TagSetID(tsNumber))
		logger.Infof("%v.%v(0x%0X): tag set %v loaded ok %v", cmdSet, tsNumber, tsNumber, ts, ok)
		if !ok {
			return fmt.Errorf("%w.%v: tag set not found", ErrSet, subCmd)
		}

		logger.Infof("%v.%v(0x%0X): tag set %v len %d", cmdSet, tsNumber, tsNumber, ts, ts.Len())
		setsStrings := make([]string, ts.Len())
		for setIdx, tagID := range ts {
			setsStrings[setIdx] = fmt.Sprintf("[%d]!!!NOT_FOUND!!!", tagID)
			tag, ok := idx.Tags().Get(tagID)
			if ok {
				setsStrings[setIdx] = string(tag.Name)
			}
		}

		fmt.Printf("%v.%v(0x%0X): %v", cmdSet, tsNumber, tsNumber, strings.Join(setsStrings, ", "))

		return nil
	}
}

func processVar(logger logging.Logger, items ...string) error {
	const subCmdID = "id"

	if len(items) < 2 {
		return fmt.Errorf("%w: at least 2 items required", ErrSet)
	}

	var subCmd = items[1]
	logger.Infof("looking for var %v", subCmd)

	collectionID, err := strconv.ParseUint(subCmd, 0, 32)
	if err != nil {
		return fmt.Errorf("%w.%v: `%v`: expected id var", ErrVar, subCmdID, subCmd)
	}

	collectionIDTyped := index.VariantID(collectionID)
	logger.Infof("looking for ID %d(0x%x)", collectionIDTyped, collectionIDTyped)
	collection, err := idx.Variants(collectionIDTyped)
	if err != nil {
		return fmt.Errorf("%w.%v: `%v`: load variant 0x%x: %v", ErrVar, subCmdID, subCmd, collectionIDTyped, err)
	}
	logger.Infof("loaded variants %v", collection)

	fmt.Printf("variant: %v: %v", collectionID, collection)
	for iterIdx, tagSetID := range collection {
		logger.Infof("unpack tagset %v(%0X)", tagSetID, tagSetID)
		tsIdx := idx.TagSetIndex()
		ts, ok := tsIdx.Get(tagSetID)
		logger.Infof("tagset %v: %v", tagSetID, ts)
		tsString := "!!!NOT FOUND!!!"
		if ok {
			tagStrings := make([]string, len(ts))
			for tagIterIdx, tagID := range ts {
				tagStrings[tagIterIdx] = "!!!NOT FOUND!!!"
				tag, tagOK := idx.Tags().Get(tagID)
				if tagOK {
					tagStrings[tagIterIdx] = string(tag.Name)
				}
			}
			tsString = fmt.Sprintf("TS%03d: %v", tagSetID, strings.Join(tagStrings, ","))
		}
		fmt.Printf("\n- %02d: %03d: %v", iterIdx, tagSetID, tsString)
	}

	fmt.Println("")

	return nil
}

func processNode(logger logging.Logger, items ...string) error {
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
		var (
			item dag.Node
			node *index.Node
		)
		if len(items) != 2 {
			return fmt.Errorf("%w.%v: 2 items required", ErrNode, subCmdInfo)
		}

		strNodeId := items[1]
		nodeID, err := strconv.Atoi(strNodeId)
		lookupByString := false
		if err != nil {
			node, err = idx.FetchItemFromParent(0, []rune(strNodeId))
			if err != nil {
				return fmt.Errorf("%w.%v: `%v`: not found", ErrNode, subCmdInfo, strNodeId)
			}
			lookupByString = true
		}

		if lookupByString {
			logger.Debugf("node %v taken by string `%v`", node.Id(), strNodeId)
			if item, err = idx.Get(node.Id()); err != nil {
				return fmt.Errorf("%w.%v: `%v`: get node: %v", ErrNode, subCmdInfo, strNodeId, err)
			}
			strNodeId = strconv.Itoa(int(node.Id()))
		} else {
			if nodeID >= idx.NodesCount() {
				return fmt.Errorf("%w.%v: `%v`: no such node", ErrNode, subCmdInfo, strNodeId)
			}

			if item, err = idx.Get(dag.ID(nodeID)); err != nil {
				return fmt.Errorf("%w.%v: `%v`: get node: %v", ErrNode, subCmdInfo, strNodeId, err)
			}
			node = idx.GetItem(dag.ID(nodeID))
			logger.Debugf("node %v taken by id ", node.Id())
		}

		word := item.Word()
		ts := item.TagSets()
		setsStrings := make([]string, len(ts))

		for tsIdx, tagSet := range ts {
			tagSetStrings := make([]string, len(tagSet))
			for tagIdx, tag := range tagSet {
				tagSetStrings[tagIdx] = string(tag.Name)
			}
			setsStrings[tsIdx] = fmt.Sprintf("\n- %02d: (%v)", tsIdx, strings.Join(tagSetStrings, ","))
		}

		fmt.Printf("Node:     %v\n", node.Id())
		fmt.Printf("Prefix:   %v\n", word)
		fmt.Printf("Parent:   %v\n", node.Item().Parent)
		if node.Item().Variants != 0 {
			fmt.Printf("Variants: %v(0x%X), %v item(s): %v\n",
				node.Item().Variants, node.Item().Variants, node.Item().Variants.TableNum(),
				strings.Join(setsStrings, ""),
			)
		} else {
			fmt.Printf("Variants: --------\n")
		}

		childrenMap := idx.GetChildrenIDMap(node.Id())
		if len(childrenMap) > 0 {
			fmt.Printf("Children: %d items:\n", len(childrenMap))
			childrenStrings := make([]string, len(childrenMap))
			currentChild := 0
			for letter, childID := range childrenMap {
				childrenStrings[currentChild] = fmt.Sprintf("%v[%d]: %d", string(letter), letter, childID)
				currentChild++
			}
			sort.Strings(childrenStrings)

			for _, childString := range childrenStrings {
				fmt.Printf("- %v\n", childString)
			}
		} else {
			fmt.Printf("Children: --------\n")
		}

		return nil
	}
}

func processReload(logger logging.Logger) (err error) {
	logger.Infof("reloading index")
	loader := opencorpora.NewLoader("")
	if idx, err = loader.LoadIndex(); err != nil {
		logger.Error("load index: %v", err)
		os.Exit(1)
	}

	return nil
}

func processInput(logger logging.Logger, line string) {
	var err error

	items := strings.Split(line, " ")

	switch items[0] {
	case cmdTag, cmdTagShort:
		err = processTag(logger, items...)
	case cmdSet, cmdSetShort:
		err = processSet(logger, items...)
	case cmdNode, cmdNodeShort:
		err = processNode(logger, items...)
	case cmdVar, cmdVarShort:
		err = processVar(logger, items...)
	case cmdReload, cmdReloadShort:
		err = processReload(logger)
	case cmdExit, cmdExitShort:
		logger.Infof("exiting")
		os.Exit(0)
	default:
		processSearch(logger, line)
	}
	fmt.Println("")

	if err != nil {
		logger.Error(err.Error())
		err = nil
	}
}

func main() {
	var (
		err    error
		logger logging.Logger
		rl     *readline.Instance
		line   string
	)

	app.InitLogging(true)

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

		processInput(logger, line)
	}

	os.Exit(0)
}
