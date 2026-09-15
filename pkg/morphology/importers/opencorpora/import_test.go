package opencorpora_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/opencorpora"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDictXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="VERB">глагол</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
  <grammeme id="gent">род. п.</grammeme>
  <grammeme id="anim">одуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme>
  <grammeme id="sing">ед. ч.</grammeme>
  <grammeme id="impf">несоверш.</grammeme>
  <grammeme id="trans">перех.</grammeme>
  <grammeme id="fem">ж. р.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот">
    <g v="nomn"/>
   </f>
   <f t="кота">
    <g v="gent"/>
   </f>
  </lemma>
  <lemma id="2" text="кот">
   <l t="кот"><g v="VERB"/><g v="impf"/><g v="trans"/></l>
   <f t="кот"/>
  </lemma>
  <lemma id="3" text="мышь">
   <l t="мышь"><g v="NOUN"/><g v="fem"/><g v="sing"/><g v="anim"/></l>
   <f t="мышь">
    <g v="nomn"/>
   </f>
   <f t="мыши">
    <g v="gent"/>
   </f>
  </lemma>
 </lemmata>
</dictionary>`

func TestImportFromXMLBasic(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.NotNil(t, d)

	assert.Equal(t, "ru", d.Language)
	require.NotNil(t, d.TagSet)
	require.Len(t, d.Words, 1, "small fixture must not overflow into more than one shard")
	require.NotNil(t, d.Words[0])

	assert.Greater(t, len(d.TagSet.Tags), 0)
	require.Len(t, d.Suffixes, 1)
	assert.Greater(t, len(d.Suffixes[0]), 0)
	require.Len(t, d.Paradigms, 1)
	assert.Greater(t, len(d.Paradigms[0]), 0)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy)
	assert.Greater(t, len(items), 0, "кот должен быть найден")
	if len(items) > 0 {
		assert.GreaterOrEqual(t, len(items[0].Values), 1, "кот имеет хотя бы один разбор")
	}
}

func TestImportFromXMLParadigmsDedup(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	require.Len(t, d.Paradigms, 1)
	assert.LessOrEqual(t, len(d.Paradigms[0]), 3, "число парадигм <= число лемм")
}

func TestImportFromXMLStemLCP(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	require.Len(t, d.Suffixes, 1)
	require.Greater(t, len(d.Suffixes[0]), 0)
	assert.Equal(t, "", d.Suffixes[0][0], "первый суффикс в шарде 0 должен быть пустым")
}

// TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes guards against the
// tag-corruption bug documented in docs/code-review-pre-1.0.md and fixed
// per docs/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md:
// each form's tag must be its lemma's own grammemes plus its own — not
// empty, not a previous form's, not an accumulating mixture.
func TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.NotNil(t, d.TagSet)

	want := []string{
		"NOUN,anim,masc,sing,nomn", // лемма "кот" (сущ.), форма "кот"
		"NOUN,anim,masc,sing,gent", // лемма "кот" (сущ.), форма "кота"
		"VERB,impf,trans",          // лемма "кот" (гл.), форма "кот" (своих граммем нет)
		"NOUN,fem,sing,anim,nomn",  // лемма "мышь", форма "мышь"
		"NOUN,fem,sing,anim,gent",  // лемма "мышь", форма "мыши"
	}
	for _, tag := range want {
		assert.Contains(t, d.TagSet.Tags, tag, "tag %q must be registered", tag)
	}

	for _, tag := range d.TagSet.Tags {
		assert.NotEqual(t, "", tag, "no form should get an empty tag")
	}
}

func TestImportFromXMLDAWGContains(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	// DAWG keys include payload suffixes, so SimilarItems is the correct lookup.
	for _, w := range []string{"кот", "кота", "мышь", "мыши"} {
		items := d.Words[0].SimilarItems(w, d.CharPolicy)
		assert.Greater(t, len(items), 0, "слово %q должно быть найдено через SimilarItems", w)
	}
}

func TestImportFromXMLRoundtrip(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy)
	require.GreaterOrEqual(t, len(items), 1)
	require.GreaterOrEqual(t, len(items[0].Values), 1, "кот имеет хотя бы один разбор")

	for _, v := range items[0].Values {
		require.Len(t, v, 4, "значение должно быть 4 байта")
	}
}

func TestImportFromXMLNoForms(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="пустая">
   <l t="пустая"><g v="NOUN"/></l>
  </lemma>
  <lemma id="2" text="есть">
   <l t="есть"><g v="NOUN"/></l>
   <f t="есть">
    <g v="nomn"/>
   </f>
  </lemma>
 </lemmata>
</dictionary>`

	d, err := opencorpora.CompileFromXML(strings.NewReader(xml), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	items := d.Words[0].SimilarItems("есть", d.CharPolicy)
	assert.Greater(t, len(items), 0, "есть должно быть найдено")

	items2 := d.Words[0].SimilarItems("пустая", d.CharPolicy)
	assert.Equal(t, 0, len(items2), "пустая не должна быть найдена")
}

// TestImportFromXMLShardsOnSuffixOverflow verifies that crossing the
// uint16 suffix-id limit (65536) makes ImportFromXML shard the
// dictionary instead of erroring or silently wrapping ids. Before
// sharding, id 65536 would wrap to 0, colliding with the very first
// suffix ever registered — see the critical finding in
// docs/code-review-pre-1.0.md and the design in
// docs/superpowers/specs/2026-09-14-suffix-sharding-design.md.
func TestImportFromXMLShardsOnSuffixOverflow(t *testing.T) {
	const uniqueSuffixes = 1 << 16 // one more than fits in a single uint16 shard

	var xml strings.Builder
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?><dictionary><lemmata>`)
	for i := 0; i < uniqueSuffixes; i++ {
		// Two forms sharing stem "слово": one bare (suffix ""), one with a
		// suffix unique to this lemma — pushes total unique suffixes past
		// the single-shard limit.
		fmt.Fprintf(&xml, `<lemma id="%d"><l t="слово"/><f t="слово"/><f t="слово%06d"/></lemma>`, i, i)
	}
	xml.WriteString(`</lemmata></dictionary>`)

	d, err := opencorpora.CompileFromXML(strings.NewReader(xml.String()), nil)
	require.NoError(t, err)
	require.Greater(t, len(d.Words), 1, "65536 unique suffixes must not fit in a single uint16 shard")
	require.Len(t, d.Suffixes, len(d.Words))
	require.Len(t, d.Paradigms, len(d.Words))

	for i, suffixes := range d.Suffixes {
		assert.LessOrEqual(t, len(suffixes), 1<<16, "shard %d exceeds the uint16 suffix-id space", i)
	}

	// Sample (not exhaustive — 65536 lemmas is already enough data)
	// wordforms across the range and confirm each is still findable in
	// whichever shard holds it.
	found := 0
	const sampleStride = 997
	for i := 0; i < uniqueSuffixes; i += sampleStride {
		word := fmt.Sprintf("слово%06d", i)
		for _, dawg := range d.Words {
			if len(dawg.SimilarItems(word, d.CharPolicy)) > 0 {
				found++
				break
			}
		}
	}
	assert.Equal(t, (uniqueSuffixes+sampleStride-1)/sampleStride, found,
		"every sampled sharded wordform must be findable in some shard")
}

func TestImportFromXMLPropertyTest(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	words := []string{"кот", "кота", "мышь", "мыши"}
	for _, w := range words {
		items := d.Words[0].SimilarItems(w, d.CharPolicy)
		assert.GreaterOrEqual(t, len(items), 1, "слово %q должно быть найдено", w)
	}
}
