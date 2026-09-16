package pymorphy

// DomainName is the data directory domain for pymorphy2 dictionaries.
const DomainName = "pymorphy"

// PyPIPackageName is the PyPI package providing pre-built pymorphy2
// dictionaries for Russian.
const PyPIPackageName = "pymorphy2-dicts-ru"

// PyPIJSONURL is the PyPI JSON API endpoint describing PyPIPackageName's
// latest release and its download URLs.
const PyPIJSONURL = "https://pypi.org/pypi/" + PyPIPackageName + "/json"

// LocalArchiveFilename is the downloaded wheel's name on disk. The wheel's
// own filename is versioned (e.g. pymorphy2_dicts_ru-2.4.417127.4579844.whl);
// it is stored under a fixed name here and the resolved version is tracked
// separately in LocalVersionFilename, so IsDownloadExists/UnpackUpdate don't
// need to parse a version out of a filename.
const LocalArchiveFilename = "pymorphy2-dicts-ru.whl"

// LocalVersionFilename records the PyPI version string of the currently
// downloaded archive, for update checks without re-querying PyPI's filename.
const LocalVersionFilename = "version.txt"

// LocalUnpackedDirName is the directory unpacked dictionary sources
// (words.dawg, paradigms.array, ...) are extracted into.
const LocalUnpackedDirName = "data"

// WheelDataSubtree is the path prefix inside the wheel archive holding the
// dictionary source files; everything else in the wheel (package __init__.py,
// metadata) is not extracted.
const WheelDataSubtree = "pymorphy2_dicts_ru/data/"

// unpackedMarkerFilename is a file expected inside LocalUnpackedDirName once
// unpacking succeeded; used to distinguish "never unpacked" from "unpacked".
const unpackedMarkerFilename = "words.dawg"
