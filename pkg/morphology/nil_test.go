package morphology_test

import (
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// TestDictionary_NilReceiver checks that every query method treats a nil
// *Dictionary as an empty dictionary instead of panicking.
func TestDictionary_NilReceiver(t *testing.T) {
	var d *morphology.Dictionary
	if got := d.Parse("кот"); got != nil {
		t.Errorf("Parse = %v", got)
	}
	if got := d.Lemma("кот"); got != nil {
		t.Errorf("Lemma = %v", got)
	}
	if d.IsKnown("кот") {
		t.Error("IsKnown = true")
	}
	if got := d.Fuzzy("кот", 1); got != nil {
		t.Errorf("Fuzzy = %v", got)
	}
	if got := d.FuzzyTop("кот", 3); got != nil {
		t.Errorf("FuzzyTop = %v", got)
	}
	if got := d.FuzzyTop("кот", 0); got != nil {
		t.Errorf("FuzzyTop(0) = %v", got)
	}
	if got := d.Language(); got != "" {
		t.Errorf("Language = %q", got)
	}
	if got := d.TagSetName(); got != "" {
		t.Errorf("TagSetName = %q", got)
	}
	if got := d.Info(); got != nil {
		t.Errorf("Info = %v", got)
	}
	if got := d.ContentHash(); got != "" {
		t.Errorf("ContentHash = %q", got)
	}
}

// TestMultiDictionary_NilMember checks that a nil member behaves as an empty
// dictionary.
func TestMultiDictionary_NilMember(t *testing.T) {
	m := morphology.NewMultiDictionary(nil)
	if m.Parse("кот") != nil || m.Lemma("кот") != nil || m.IsKnown("кот") ||
		m.Fuzzy("кот", 1) != nil || m.FuzzyTop("кот", 3) != nil || m.Close() != nil {
		t.Error("a nil member must behave as an empty dictionary")
	}
}

// TestSentinelErrorPrefix checks that every exported sentinel error names
// the package, like the wrapped errors do.
func TestSentinelErrorPrefix(t *testing.T) {
	for _, err := range []error{
		morphology.ErrNoEntries,
		morphology.ErrBuilderClosed,
		morphology.ErrIncompatibleDictionaries,
		morphology.ErrPredictionSharded,
	} {
		if !strings.HasPrefix(err.Error(), "morphology: ") {
			t.Errorf("%q lacks the \"morphology: \" prefix", err)
		}
	}
}
