package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/chzyer/readline"
)

const usage = `gomorphy — морфологический анализ слов по словарям в едином формате (GMOR).

Usage:
  gomorphy [flags] <command> [args]
  gomorphy [flags]              interactive console (with -dict)
  gomorphy import pymorphy2 <dir> -o <file.dat>
  gomorphy import opencorpora <dict.xml> -o <file.dat>

Commands:
  import     pymorphy2 <dir>       compile pymorphy2 dict directory to GMOR file
  import     opencorpora <xml>     compile OpenCorpora dict.xml to GMOR file
  lookup     <word>               exact dictionary lookup (all readings)
  lemmas     <word>               find initial forms (lemmas)
  fuzzy      <word> [maxDist]     fuzzy search within edit distance (default 2)
  top        <word> <N>           N nearest words by edit distance

Flags:
  -dict <path>   path to compiled dictionary (.dat file, GMOR)
  -o <path>      output file for import

Console:
  Run without a command to enter interactive mode.
  TAB — autocomplete commands, ENTER — execute, type exit/quit to leave.
`

var commands = []string{"lookup", "lemmas", "fuzzy", "top", "import"}

type commandCompleter struct{}

func (c *commandCompleter) Do(line []rune, pos int) ([][]rune, int) {
	input := string(line[:pos])
	parts := strings.Fields(input)

	completeCmd := len(parts) == 0 || (len(parts) == 1 && !strings.HasSuffix(input, " "))
	if !completeCmd {
		return nil, 0
	}

	prefix := ""
	if len(parts) == 1 {
		prefix = parts[0]
	}

	var result [][]rune
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, prefix) {
			suffix := cmd[len(prefix):]
			result = append(result, []rune(suffix))
		}
	}

	return result, len(prefix)
}

func main() {
	dictPath := flag.String("dict", "", "path to compiled dictionary (.dat)")
	outPath := flag.String("o", "", "output file for import")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if flag.Arg(0) == "import" {
		importArgs := flag.Args()[1:]
		out := *outPath
		if o := findOutFlag(importArgs); o != "" {
			out = o
		}
		runImport(stripOutFlag(importArgs), out)
		return
	}

	if *dictPath == "" {
		fmt.Fprintln(os.Stderr, "error: -dict flag is required")
		os.Exit(1)
	}

	d, err := morphology.Open(*dictPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open dictionary: %v\n", err)
		os.Exit(1)
	}

	defer func() { _ = d.Close() }()

	if flag.NArg() == 0 {
		runConsole(d)
		return
	}

	cmd, args := flag.Arg(0), flag.Args()[1:]

	switch cmd {
	case "lookup":
		if err := runLookup(d, args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "lemmas":
		if err := runLemmas(d, args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "fuzzy":
		if err := runFuzzy(d, args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "top":
		if err := runTop(d, args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

// findOutFlag ищет -o <path> (или -o=<path>) в positional args — flag-пакет
// останавливает парсинг на первом нефлаге, поэтому после подкоманды
// флаги приходится доставать вручную.
func findOutFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], "-o=") {
			return strings.TrimPrefix(args[i], "-o=")
		}
	}
	return ""
}

func stripOutFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "-o=") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func runImport(args []string, out string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "error: usage: gomorphy import <source> <path> -o <out.dat>")
		os.Exit(1)
	}

	source := args[0]
	path := args[1]

	switch source {
	case "pymorphy2":
		d, err := morphology.OpenPyMorphy(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: import pymorphy2: %v\n", err)
			os.Exit(1)
		}
		if err := d.SaveTo(out); err != nil {
			fmt.Fprintf(os.Stderr, "error: write %s: %v\n", out, err)
			os.Exit(1)
		}
		fmt.Printf("saved %s\n", out)

	case "opencorpora":
		d, err := morphology.CompileFromXMLFile(path, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: import opencorpora: %v\n", err)
			os.Exit(1)
		}
		if err := d.SaveTo(out); err != nil {
			fmt.Fprintf(os.Stderr, "error: write %s: %v\n", out, err)
			os.Exit(1)
		}
		fmt.Printf("saved %s\n", out)

	default:
		fmt.Fprintf(os.Stderr, "error: unknown import source %q (use pymorphy2 or opencorpora)\n", source)
		os.Exit(1)
	}
}

func runConsole(d *morphology.Dictionary) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "gomorphy> ",
		AutoComplete:    &commandCompleter{},
		HistoryFile:     "",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: init console: %v\n", err)
		os.Exit(1)
	}

	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}

			if err == io.EOF {
				fmt.Println()
				break
			}

			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			break
		}

		fields := strings.Fields(line)
		cmd, args := fields[0], fields[1:]

		switch cmd {
		case "lookup":
			if err := runLookup(d, args); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		case "lemmas":
			if err := runLemmas(d, args); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		case "fuzzy":
			if err := runFuzzy(d, args); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		case "top":
			if err := runTop(d, args); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		default:
			fmt.Fprintf(os.Stderr, "error: unknown command %q\n", cmd)
		}
	}
}

func runLookup(d *morphology.Dictionary, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lookup <word>")
	}

	word := args[0]
	readings := d.Parse(word)
	if len(readings) == 0 {
		return fmt.Errorf("lookup %q: no readings", word)
	}

	for _, r := range readings {
		fmt.Printf("%s\t%s\t%s\tpara#%d\n", r.Word, r.Normal, r.Tag, r.Para)
	}

	return nil
}

func runLemmas(d *morphology.Dictionary, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lemmas <word>")
	}

	word := args[0]
	refs := d.Lemma(word)
	if len(refs) == 0 {
		return fmt.Errorf("lemmas %q: no lemmas", word)
	}

	for _, r := range refs {
		fmt.Printf("%s\t%s\n", r.Normal, r.Tag)
	}

	return nil
}

func runFuzzy(d *morphology.Dictionary, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: fuzzy <word> [maxDist]")
	}

	word := args[0]
	maxDist := 2

	if len(args) == 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 0 {
			return fmt.Errorf("invalid maxDist %q", args[1])
		}

		maxDist = n
	}

	matches := d.Fuzzy(word, maxDist)
	for _, m := range matches {
		fmt.Printf("%d\t%s\n", m.Distance, m.Word)
	}

	return nil
}

func runTop(d *morphology.Dictionary, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: top <word> <N>")
	}

	word := args[0]
	n, err := strconv.Atoi(args[1])
	if err != nil || n <= 0 {
		return fmt.Errorf("invalid N %q", args[1])
	}

	matches := d.FuzzyTop(word, n)
	for _, m := range matches {
		fmt.Printf("%d\t%s\n", m.Distance, m.Word)
	}

	return nil
}
