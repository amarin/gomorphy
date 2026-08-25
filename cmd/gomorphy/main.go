package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/amarin/gomorphy/pkg/dictionary"
)

const usage = `gomorphy — morphological analysis of Russian words via OpenCorpora dictionary.

Usage:
  gomorphy [flags] <command> [args]

Commands:
  lookup   <word>              exact dictionary lookup
  lemmas   <word>              find initial forms (lemmas)
  fuzzy    <word> [maxDist]    fuzzy search within edit distance (default 2)
  top      <word> <N>          N nearest words by edit distance

Flags:
  -dict <path>   path to compiled dictionary (.dat file)
`

func main() {
	dictPath := flag.String("dict", "", "path to compiled dictionary (.dat)")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cmd, args := flag.Arg(0), flag.Args()[1:]

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

	switch cmd {
	case "lookup":
		runLookup(d, args)
	case "lemmas":
		runLemmas(d, args)
	case "fuzzy":
		runFuzzy(d, args)
	case "top":
		runTop(d, args)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

func runLookup(d *dictionary.Dictionary, args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: gomorphy lookup <word>")
		os.Exit(1)
	}

	word := args[0]
	forms, err := d.Lookup(word)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lookup %q: %v\n", word, err)
		os.Exit(1)
	}

	for _, f := range forms {
		fmt.Printf("%s\t%s\t lemma#%d\n", f.Text, f.Ancode, f.LemmaID)
	}
}

func runLemmas(d *dictionary.Dictionary, args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: gomorphy lemmas <word>")
		os.Exit(1)
	}

	word := args[0]
	refs, err := d.Lemmas(word)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lemmas %q: %v\n", word, err)
		os.Exit(1)
	}

	for _, r := range refs {
		fmt.Printf("#%d\t%s\t%s\n", r.ID, r.Text, strings.Join(r.Grammemes, ","))
	}
}

func runFuzzy(d *dictionary.Dictionary, args []string) {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(os.Stderr, "usage: gomorphy fuzzy <word> [maxDist]")
		os.Exit(1)
	}

	word := args[0]
	maxDist := 2

	if len(args) == 2 {
		n := 0
		if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil || n < 0 {
			fmt.Fprintf(os.Stderr, "error: invalid maxDist %q\n", args[1])
			os.Exit(1)
		}

		maxDist = n
	}

	matches, err := d.Fuzzy(word, maxDist)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fuzzy %q: %v\n", word, err)
		os.Exit(1)
	}

	for _, m := range matches {
		fmt.Printf("%d\t%s\n", m.Distance, m.Text)
	}
}

func runTop(d *dictionary.Dictionary, args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: gomorphy top <word> <N>")
		os.Exit(1)
	}

	word := args[0]
	n := 0
	if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil || n <= 0 {
		fmt.Fprintf(os.Stderr, "error: invalid N %q\n", args[1])
		os.Exit(1)
	}

	matches, err := d.FuzzyTop(word, n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "top %q: %v\n", word, err)
		os.Exit(1)
	}

	for _, m := range matches {
		fmt.Printf("%d\t%s\n", m.Distance, m.Text)
	}
}
