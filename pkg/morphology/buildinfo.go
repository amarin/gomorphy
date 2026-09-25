package morphology

import "time"

// BuildInfo — diagnostic metadata of a dictionary (the "info" section of a
// GMOR file): when and with what it was built, where the data came from. No
// field is required for Parse/Lemma/Fuzzy to work — all fields are optional.
type BuildInfo struct {
	BuiltAt        time.Time // set by SaveTo
	LibraryVersion string    // set by SaveTo: the Version that wrote the file
	Source         string    // set by the importer, Builder, ImportTSV or Merge
	SourceVersion  string    // version of the source data, when known
	Author         string
	Description    string
	SourceURL      string
}

// Info returns the dictionary's diagnostic metadata. It is nil only for a
// nil dictionary or a GMOR file written before the "info" section existed.
// Every importer, Builder, ImportTSV and Merge set at least Source
// ("pymorphy2", "opencorpora", "unimorph", "builder", "tsv", "merge");
// BuiltAt and LibraryVersion are filled in by SaveTo.
func (x *Dictionary) Info() *BuildInfo {
	if x == nil || x.d == nil || x.d.Info == nil {
		return nil
	}
	info := *x.d.Info
	return &BuildInfo{
		BuiltAt:        info.BuiltAt,
		LibraryVersion: info.LibraryVersion,
		Source:         info.Source,
		SourceVersion:  info.SourceVersion,
		Author:         info.Author,
		Description:    info.Description,
		SourceURL:      info.SourceURL,
	}
}
