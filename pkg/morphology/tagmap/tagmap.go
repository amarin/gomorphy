package tagmap

// source pairs a tokenizer with its mapping table for one dictionary
// format (one TagSet.Name).
type source struct {
	tokenize func(tag string) []string
	table    map[string]Feature
}

// sources is the registry of dictionary formats this package knows how
// to normalize, keyed by TagSet.Name.
var sources = map[string]source{
	"opencorpora":     {tokenize: tokenizeOpenCorpora, table: openCorporaTable},
	"opencorpora-int": {tokenize: tokenizeOpenCorporaInt, table: openCorporaIntTable},
	"unimorph":        {tokenize: tokenizeUniMorph, table: unimorphTable},
}

// Map normalizes tag (as found in a Dictionary's TagSet, i.e. a value
// TagSet.TagName would return) into a Bundle, using dictName (the
// dictionary's TagSet.Name) to pick the tokenizer and mapping table.
//
// ok is false only when dictName is not a source this package knows
// about at all — a caller bug (asking for an unregistered source), not a
// property of tag's content. Any other input always returns ok=true;
// tokens absent from the source's table land in Bundle.Unmapped rather
// than causing an error (see Bundle's doc comment).
func Map(dictName, tag string) (Bundle, bool) {
	src, known := sources[dictName]
	if !known {
		return Bundle{}, false
	}
	return buildBundle(src.tokenize(tag), src.table), true
}

// Known reports whether dictName (a dictionary's TagSet.Name) is a
// source this package can normalize — i.e. whether Map(dictName, …)
// returns ok=true.
func Known(dictName string) bool {
	_, ok := sources[dictName]
	return ok
}
