package morphology

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// tsvTagSetName is the default TagSet name for dictionaries built through
// ImportTSV.
const tsvTagSetName = "tsv"

// ImportTSV reads a tab-separated stream of wordform entries from r and
// builds an immutable *Dictionary through the same shared pipeline as
// Builder (dense alphabet, prediction rebuilt). It is a package-level
// function rather than a Builder method because it is not a streaming
// consumer: it accumulates the whole stream and hands it to the shared
// buildFromEntries helper in one call.
//
// Stream format, one entry per line:
//
//	lemma[TAB]wordform[<TAB>tags]
//
// Blank lines and lines whose first non-space byte is '#' are skipped.
// Leading/trailing whitespace is trimmed from each field (tab is the only
// delimiter; stray spaces around fields are tolerated). A missing lemma
// makes the wordform its own lemma (auto-lemma); a missing tags column is
// "" — matching Builder.AddForm. An empty wordform column, or any other
// column count — 1, or more than 3 — is an error naming the offending line
// number.
//
// The resulting dictionary reports Source "tsv" when the caller leaves
// opts.Source empty (mirroring how Builder defaults Source to "builder"); a
// caller-supplied opts.Source is honored as-is. Input is never lower-cased
// and tags are never normalized; one entry per line is fully general.
func ImportTSV(r io.Reader, opts BuilderOptions) (*Dictionary, error) {
	if opts.Source == "" {
		opts.Source = "tsv"
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20) // 1MB max line: compound/hyphenated tokens

	var entries []internal.BuildEntry
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Split(line, "\t")

		var lemma, word, tag string
		switch len(fields) {
		case 2:
			lemma, word, tag = strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), ""
		case 3:
			lemma, word, tag = strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), strings.TrimSpace(fields[2])
		default:
			return nil, fmt.Errorf("morphology: tsv: line %d: expected lemma<TAB>wordform[<TAB>tags], got %d columns", lineNum, len(fields))
		}

		if word == "" {
			return nil, fmt.Errorf("morphology: tsv: line %d: wordform must not be empty", lineNum)
		}
		if lemma == "" {
			lemma = word
		}
		entries = append(entries, internal.BuildEntry{Word: word, Lemma: lemma, Tag: tag})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("morphology: tsv: scan: %w", err)
	}

	return buildFromEntries(opts, entries, tsvTagSetName)
}
