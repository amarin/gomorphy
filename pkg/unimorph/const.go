package unimorph

// DomainName is the data directory domain for UniMorph dictionaries.
const DomainName = "unimorph"

// LocalUnpackedFilename is the downloaded TSV's name on disk. There is
// no archive to unpack for this source (see Loader's doc comment) — the
// downloaded file already has this name, "unpacked" or not.
const LocalUnpackedFilename = "data"

// isoCodes maps gomorphy's own language code (as used everywhere else in
// this codebase — internal.Dictionary.Language, RussianCharPolicy, the
// CLI's -lang flag) to UniMorph's own ISO 639-3 repository/file name.
// The two conventions differ (gomorphy: "ru", UniMorph: "rus", verified
// against github.com/unimorph/rus) — only "ru" is supported today;
// extending to more languages means adding entries here. See
// docs/en/research/0006-unimorph-import-plan.md, Q1.
var isoCodes = map[string]string{
	"ru": "rus",
}

// remoteURL returns the raw-file download URL for a UniMorph ISO 639-3
// code (e.g. "rus"), matching the layout of every github.com/unimorph/*
// repository: a single file named after the language, on the master
// branch.
func remoteURL(iso string) string {
	return "https://raw.githubusercontent.com/unimorph/" + iso + "/master/" + iso
}
