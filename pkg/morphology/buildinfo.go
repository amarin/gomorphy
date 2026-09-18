package morphology

import "time"

// BuildInfo — diagnostic metadata of a dictionary (the "info" section of a
// GMOR file): when and with what it was built, where the data came from. No
// field is required for Parse/Lemma/Fuzzy to work — all fields are optional.
type BuildInfo struct {
	BuiltAt        time.Time
	LibraryVersion string
	Source         string
	SourceVersion  string
	Author         string
	Description    string
	SourceURL      string
}

// Info returns the dictionary's diagnostic metadata, if present in the
// file. nil is returned for dictionaries without an "info" section: files
// built before it was introduced, or dictionaries assembled manually via
// the Builder API and never passed through SaveTo.
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
