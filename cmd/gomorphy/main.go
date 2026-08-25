package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/amarin/gomorphy/pkg/dictionary"
	"github.com/chzyer/readline"
)

const usage = `gomorphy — morphological analysis of Russian words via OpenCorpora dictionary.

Usage:
  gomorphy [flags] <command> [args]
  gomorphy [flags]              interactive console (with -dict)

Commands:
  lookup   <word>              exact dictionary lookup
  lemmas   <word>              find initial forms (lemmas)
  fuzzy    <word> [maxDist]    fuzzy search within edit distance (default 2)
  top      <word> <N>          N nearest words by edit distance

Flags:
  -dict <path>   path to compiled dictionary (.dat file)

Console:
  Run without a command to enter interactive mode.
  TAB — autocomplete commands, ENTER — execute, type exit/quit to leave.
`

var commands = []string{"lookup", "lemmas", "fuzzy", "top"}

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
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if *dictPath == "" {
		fmt.Fprintln(os.Stderr, "error: -dict flag is required")
		os.Exit(1)
	}

	d, err := dictionary.Open(*dictPath)
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

func runConsole(d *dictionary.Dictionary) {
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

func runLookup(d *dictionary.Dictionary, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lookup <word>")
	}

	word := args[0]
	forms, err := d.Lookup(word)
	if err != nil {
		return fmt.Errorf("lookup %q: %w", word, err)
	}

	for _, f := range forms {
		fmt.Printf("%s\t%s\t lemma#%d\n", f.Text, f.Ancode, f.LemmaID)
	}

	return nil
}

func runLemmas(d *dictionary.Dictionary, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lemmas <word>")
	}

	word := args[0]
	refs, err := d.Lemmas(word)
	if err != nil {
		return fmt.Errorf("lemmas %q: %w", word, err)
	}

	for _, r := range refs {
		fmt.Printf("#%d\t%s\t%s\n", r.ID, r.Text, strings.Join(r.Grammemes, ","))
	}

	return nil
}

func runFuzzy(d *dictionary.Dictionary, args []string) error {
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

	matches, err := d.Fuzzy(word, maxDist)
	if err != nil {
		return fmt.Errorf("fuzzy %q: %w", word, err)
	}

	for _, m := range matches {
		fmt.Printf("%d\t%s\n", m.Distance, m.Text)
	}

	return nil
}

func runTop(d *dictionary.Dictionary, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: top <word> <N>")
	}

	word := args[0]
	n, err := strconv.Atoi(args[1])
	if err != nil || n <= 0 {
		return fmt.Errorf("invalid N %q", args[1])
	}

	matches, err := d.FuzzyTop(word, n)
	if err != nil {
		return fmt.Errorf("top %q: %w", word, err)
	}

	for _, m := range matches {
		fmt.Printf("%d\t%s\n", m.Distance, m.Text)
	}

	return nil
}
