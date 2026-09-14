package opencorpora_test

import (
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
   <l g="NOUN,anim,masc,sing">
    <f t="кот">
     <g v="nomn"/>
    </f>
    <f t="кота">
     <g v="gent"/>
    </f>
   </l>
  </lemma>
  <lemma id="2" text="кот">
   <l g="VERB,impf,trans">
    <f t="кот"/>
   </l>
  </lemma>
  <lemma id="3" text="мышь">
   <l g="NOUN,fem,sing,anim">
    <f t="мышь">
     <g v="nomn"/>
    </f>
    <f t="мыши">
     <g v="gent"/>
    </f>
   </l>
  </lemma>
 </lemmata>
</dictionary>`

func TestImportFromXMLBasic(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.NotNil(t, d)

	assert.Equal(t, "ru", d.Language)
	require.NotNil(t, d.TagSet)
	require.NotNil(t, d.Words)

	assert.Greater(t, len(d.TagSet.Tags), 0)
	assert.Greater(t, len(d.Suffixes), 0)
	assert.Greater(t, len(d.Paradigms), 0)

	items := d.Words.SimilarItems("кот", d.CharPolicy)
	assert.Greater(t, len(items), 0, "кот должен быть найден")
	if len(items) > 0 {
		assert.GreaterOrEqual(t, len(items[0].Values), 1, "кот имеет хотя бы один разбор")
	}
}

func TestImportFromXMLParadigmsDedup(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(d.Paradigms), 3, "число парадигм <= число лемм")
}

func TestImportFromXMLStemLCP(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	require.Greater(t, len(d.Suffixes), 0)
	assert.Equal(t, "", d.Suffixes[0], "первый суффикс должен быть пустым")
}

func TestImportFromXMLDAWGContains(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	// DAWG keys include payload suffixes, so SimilarItems is the correct lookup.
	for _, w := range []string{"кот", "кота", "мышь", "мыши"} {
		items := d.Words.SimilarItems(w, d.CharPolicy)
		assert.Greater(t, len(items), 0, "слово %q должно быть найдено через SimilarItems", w)
	}
}

func TestImportFromXMLRoundtrip(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	items := d.Words.SimilarItems("кот", d.CharPolicy)
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
   <l g="NOUN">
   </l>
  </lemma>
  <lemma id="2" text="есть">
   <l g="NOUN">
    <f t="есть">
     <g v="nomn"/>
    </f>
   </l>
  </lemma>
 </lemmata>
</dictionary>`

	d, err := opencorpora.CompileFromXML(strings.NewReader(xml), nil)
	require.NoError(t, err)

	items := d.Words.SimilarItems("есть", d.CharPolicy)
	assert.Greater(t, len(items), 0, "есть должно быть найдено")
	
	items2 := d.Words.SimilarItems("пустая", d.CharPolicy)
	assert.Equal(t, 0, len(items2), "пустая не должна быть найдена")
}

func TestImportFromXMLPropertyTest(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	words := []string{"кот", "кота", "мышь", "мыши"}
	for _, w := range words {
		items := d.Words.SimilarItems(w, d.CharPolicy)
		assert.GreaterOrEqual(t, len(items), 1, "слово %q должно быть найдено", w)
	}
}
