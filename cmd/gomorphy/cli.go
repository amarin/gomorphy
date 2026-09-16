package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

var consoleCommands = []string{"lookup", "lemmas", "fuzzy", "top", "exit", "quit"}

// consoleCompleter is cli.go's own readline completer. It is named
// distinctly from the old cmd/gomorphy/main.go's package-scope
// commandCompleter (same package, still present until Task 6 removes it)
// to avoid a duplicate-declaration compile error; Task 6's rewrite of
// main.go drops the old declaration entirely, so this name has no
// external significance beyond this file.
type consoleCompleter struct{}

func (c *consoleCompleter) Do(line []rune, pos int) ([][]rune, int) {
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
	for _, cmd := range consoleCommands {
		if strings.HasPrefix(cmd, prefix) {
			suffix := cmd[len(prefix):]
			result = append(result, []rune(suffix))
		}
	}
	return result, len(prefix)
}

func newCLICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cli",
		Short: "interactive console",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return runInteractiveConsole(cmd.OutOrStdout(), m)
		},
	}
}

// runInteractiveConsole is named distinctly from the old main.go's
// package-scope runConsole (same reason as consoleCompleter above).
func runInteractiveConsole(out io.Writer, m *morphology.MultiDictionary) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "gomorphy> ",
		AutoComplete:    &consoleCompleter{},
		HistoryFile:     "",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("init console: %w", err)
	}
	defer func() { _ = rl.Close() }()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				_, _ = fmt.Fprintln(out)
				return nil
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}

		fields := strings.Fields(line)
		cmd, cmdArgs := fields[0], fields[1:]

		var runErr error
		switch cmd {
		case "lookup":
			runErr = requireOneArg(cmdArgs, func(word string) error { return doLookup(out, m, word) })
		case "lemmas":
			runErr = requireOneArg(cmdArgs, func(word string) error { return doLemmas(out, m, word) })
		case "fuzzy":
			runErr = runConsoleFuzzy(out, m, cmdArgs)
		case "top":
			runErr = runConsoleTop(out, m, cmdArgs)
		default:
			runErr = fmt.Errorf("unknown command %q", cmd)
		}
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", runErr)
		}
	}
}

func requireOneArg(args []string, fn func(string) error) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: <word>")
	}
	return fn(args[0])
}

func runConsoleFuzzy(out io.Writer, m *morphology.MultiDictionary, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: fuzzy <word> [maxDist]")
	}
	maxDist := 2
	if len(args) == 2 {
		n, err := parseNonNegativeInt(args[1])
		if err != nil {
			return err
		}
		maxDist = n
	}
	return doFuzzy(out, m, args[0], maxDist)
}

func runConsoleTop(out io.Writer, m *morphology.MultiDictionary, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: top <word> <N>")
	}
	n, err := parseNonNegativeInt(args[1])
	if err != nil || n == 0 {
		return fmt.Errorf("invalid N %q", args[1])
	}
	return doTop(out, m, args[0], n)
}
