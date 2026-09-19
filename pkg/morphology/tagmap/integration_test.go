//go:build integration

// Integration tests here require a real OpenCorpora dict.xml, a real
// pymorphy2 data directory, and/or a real UniMorph rus TSV on disk.
// Run explicitly:
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
	unimorphTSVEnvVar  = "GOMORPHY_UNIMORPH_TSV"
	unimorphTSVDefault = ".data/unimorph/ru/data"
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

// findUniMorphTSV mirrors findDictXML for the UniMorph rus TSV (same
// search: env var override, else walk up from the working directory
// looking for .data/unimorph/ru/data).
func findUniMorphTSV() string {
	if p := os.Getenv(unimorphTSVEnvVar); p != "" {
		return p
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, unimorphTSVDefault)
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

// TestMapUniMorphAgreesOnSharedFeaturesWithRealOpenCorpora checks
// tagmap.Map("unimorph", ...) against a real compiled UniMorph
// dictionary, cross-checked against OpenCorpora (when available) on
// the features UniMorph's own rus data actually carries.
//
// Unlike TestMapAgreesAcrossRealOpenCorporaAndPymorphy2Dictionaries,
// this isn't a full Bundle.Features equality check: the real rus TSV
// doesn't tag "кот"'s nominative singular with animacy or gender at
// all ("N;NOM;SG" — verified against the live download, see
// docs/en/implementation/stage-16-import-unimorph.md), while
// OpenCorpora's tag for the same word carries both
// ("NOUN,anim,masc,sing,nomn"). That's a real difference in what each
// source's data records, not a mapping bug — so this test only
// asserts agreement on the dimensions UniMorph's tag actually has
// (part of speech, case, number).
func TestMapUniMorphAgreesOnSharedFeaturesWithRealOpenCorpora(t *testing.T) {
	tsvPath := findUniMorphTSV()
	if tsvPath == "" {
		t.Skipf("UniMorph rus TSV not found (%s to override)", unimorphTSVEnvVar)
	}

	umDict, err := morphology.CompileFromUniMorphFile(tsvPath, morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)
	defer func() { _ = umDict.Close() }()

	var umTag string
	for _, r := range umDict.Parse("кот") {
		if strings.Contains(r.Tag, "N") && strings.Contains(r.Tag, "NOM") && strings.Contains(r.Tag, "SG") {
			umTag = r.Tag
			break
		}
	}
	require.NotEmpty(t, umTag, "no N/NOM/SG reading found for \"кот\" in the real rus dictionary")

	umBundle, ok := tagmap.Map("unimorph", umTag)
	require.True(t, ok)
	assert.Empty(t, umBundle.Unmapped, "unimorph tag %q has tokens outside the representative table", umTag)
	assert.Equal(t, []tagmap.Feature{
		{Dim: tagmap.DimPartOfSpeech, Value: "N"},
		{Dim: tagmap.DimCase, Value: "NOM"},
		{Dim: tagmap.DimNumber, Value: "SG"},
	}, umBundle.Features)

	xmlPath := findDictXML()
	if xmlPath == "" {
		t.Skipf("dict.xml not found (%s to override), skipping the OpenCorpora cross-check", dictXMLEnvVar)
	}
	ocDict, err := morphology.CompileFromXMLFile(xmlPath, nil)
	require.NoError(t, err)
	defer func() { _ = ocDict.Close() }()

	ocTag := nominativeSingularNounTag(t, ocDict.Parse("кот"))
	ocBundle, ok := tagmap.Map("opencorpora", ocTag)
	require.True(t, ok)

	// OpenCorpora's bundle must contain every feature UniMorph's does
	// (a superset, not equality — see the doc comment above).
	for _, f := range umBundle.Features {
		assert.Contains(t, ocBundle.Features, f,
			"opencorpora tag %q missing feature %+v present in unimorph tag %q", ocTag, f, umTag)
	}
}
