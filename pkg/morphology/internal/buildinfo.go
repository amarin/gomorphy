package internal

import (
	"encoding/json"
	"time"
)

// BuildInfo — diagnostic metadata of the dictionary: the "info" section in
// the GMOR file. Unlike meta (language + CharPolicy — required for Parse),
// no BuildInfo field is required for the dictionary to work: all fields
// are optional (zero value = "not specified"), and the section as a whole
// may be absent entirely (files built before this field existed, or
// dictionaries assembled manually via the Builder API without calling
// SaveTo).
type BuildInfo struct {
	// BuiltAt and LibraryVersion are set by SaveTo itself on every save —
	// values set beforehand by the importer are overwritten.
	BuiltAt        time.Time `json:"built_at,omitempty"`
	LibraryVersion string    `json:"library_version,omitempty"`

	// Source — the data source: "opencorpora", "pymorphy2", "unimorph",
	// "tsv", ... Filled in by the importer when building the Dictionary.
	Source string `json:"source,omitempty"`

	// SourceVersion — the version/revision of the source data (e.g., for
	// OpenCorpora dict.xml — the version/revision attributes of the root
	// tag). Not yet filled in by any importer: xmlscan does not expose the
	// root tag's attributes — see docs/en/todo.md.
	SourceVersion string `json:"source_version,omitempty"`

	// Author, Description — free-form fields, primarily relevant for
	// thematic dictionaries (see docs/en/todo.md, Stage 19).
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`

	// SourceURL — where to manually (or via an agent) download a fresh
	// version of the dictionary. A placeholder for a future version-list
	// standard — see docs/en/todo.md.
	SourceURL string `json:"source_url,omitempty"`
}

// EncodeBuildInfo serializes BuildInfo to JSON for the "info" section.
func EncodeBuildInfo(info *BuildInfo) []byte {
	if info == nil {
		info = &BuildInfo{}
	}
	data, _ := json.Marshal(info)
	return data
}

// DecodeBuildInfo reads a BuildInfo written by EncodeBuildInfo.
func DecodeBuildInfo(data []byte) (*BuildInfo, error) {
	var info BuildInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, wrap(ErrMalformedFile, "info")
	}
	return &info, nil
}
