//go:build integration

// Integration test here requires a real OpenCorpora dict.xml and a real
// pymorphy2 data directory on disk. Run explicitly:
//
//	go test -tags=integration ./pkg/morphology/tagmap/... -v
package tagmap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dictXMLEnvVar      = "GOMORPHY_DICT_XML"
	dictXMLDefaultPath = ".data/opencorpora/dict.xml"
	pymorphyDirEnvVar  = "GOMORPHY_PYMORPHY2_DIR"
)

// findDictXML mirrors
// pkg/morphology/importers/opencorpora/real_dict_integration_test.go's
// findRealDictXML: walks up from the working directory looking for
// .data/opencorpora/dict.xml, so this test passes both from the package
// dir and from the repo root.
func findDictXML() string {
	if p := os.Getenv(dictXMLEnvVar); p != "" {
		return p
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, dictXMLDefaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// nominativeSingularNounTag finds, among readings, the one whose tag
// looks like a nominative singular noun — robust to the two different
// tag syntaxes (opencorpora vs opencorpora-int both use these exact
// substrings, just arranged differently) without needing to know which
// dictionary produced readings.
func nominativeSingularNounTag(t *testing.T, readings []morphology.Reading) string {
	t.Helper()
	for _, r := range readings {
		if strings.Contains(r.Tag, "NOUN") &&
			strings.Contains(r.Tag, "nomn") &&
			strings.Contains(r.Tag, "sing") {
			return r.Tag
		}
	}
	t.Fatalf("no NOUN/nomn/sing reading found among %d readings", len(readings))
	return ""
}

func TestMapAgreesAcrossRealOpenCorporaAndPymorphy2Dictionaries(t *testing.T) {
	xmlPath := findDictXML()
	if xmlPath == "" {
		t.Skipf("dict.xml not found (%s to override)", dictXMLEnvVar)
	}
	pymorphyDir := os.Getenv(pymorphyDirEnvVar)
	if pymorphyDir == "" {
		t.Skipf("%s not set", pymorphyDirEnvVar)
	}

	ocDict, err := morphology.CompileFromXMLFile(xmlPath, nil)
	require.NoError(t, err)
	defer func() { _ = ocDict.Close() }()

	pmDict, err := morphology.OpenPyMorphy(pymorphyDir)
	require.NoError(t, err)
	defer func() { _ = pmDict.Close() }()

	ocTag := nominativeSingularNounTag(t, ocDict.Parse("кот"))
	pmTag := nominativeSingularNounTag(t, pmDict.Parse("кот"))

	ocBundle, ok := tagmap.Map("opencorpora", ocTag)
	require.True(t, ok)
	pmBundle, ok := tagmap.Map("opencorpora-int", pmTag)
	require.True(t, ok)

	assert.Equal(t, ocBundle.Features, pmBundle.Features,
		"opencorpora tag %q and pymorphy2 tag %q should normalize to the same universal bundle",
		ocTag, pmTag)
	assert.Empty(t, ocBundle.Unmapped,
		"opencorpora tag %q has grammemes outside the representative table", ocTag)
	assert.Empty(t, pmBundle.Unmapped,
		"pymorphy2 tag %q has grammemes outside the representative table", pmTag)
}
