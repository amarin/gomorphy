package morphology

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// shardedPredictionDict is a dense two-shard dictionary with sharded
// prediction: shard 0 holds кошка/кошки, shard 1 окно/окна, each as its
// shard's paradigm 0 with the same tags.
func shardedPredictionDict(t *testing.T) *Dictionary {
	t.Helper()
	build := func(entries ...internal.BuildEntry) *internal.Dictionary {
		d, err := internal.BuildDictionaryFromEntries(internal.BuildOptions{
			Language: "ru", CharPolicy: internal.RussianCharPolicy(), TagSetName: "tsv",
		}, entries)
		require.NoError(t, err)
		return d
	}
	a := build(
		internal.BuildEntry{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		internal.BuildEntry{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
	)
	b := build(
		internal.BuildEntry{Word: "окно", Lemma: "окно", Tag: "NOUN,anim,femn,sing,nomn"},
		internal.BuildEntry{Word: "окна", Lemma: "окно", Tag: "NOUN,anim,femn,sing,gent"},
	)
	d := internal.NewDictionary("ru", a.TagSet,
		[][]string{a.Suffixes[0], b.Suffixes[0]}, a.Prefixes,
		[][]internal.Paradigm{a.Paradigms[0], b.Paradigms[0]},
		[]*internal.DAWG{a.Words[0], b.Words[0]}, internal.RussianCharPolicy())
	require.NoError(t, internal.BuildPrediction(d, productive))
	require.True(t, d.PredictionSharded)
	require.NoError(t, internal.RecompileDense(d))
	return &Dictionary{d: d}
}

// predictedFrom reports whether Parse(word) has a predicted reading with
// this shard and lemma, returning it.
func predictedFrom(d *Dictionary, word string, shard int, normal string) (Reading, bool) {
	for _, r := range d.Parse(word) {
		if r.Predicted && r.Shard == shard && r.Normal == normal {
			return r, true
		}
	}
	return Reading{}, false
}

func TestParsePredictsAcrossShards(t *testing.T) {
	d := shardedPredictionDict(t)

	r, ok := predictedFrom(d, "бревна", 1, "бревно")
	require.True(t, ok, "бревна is predicted from окна (shard 1): %+v", d.Parse("бревна"))
	assert.Equal(t, "NOUN,anim,femn,sing,gent", r.Tag)

	_, ok = predictedFrom(d, "мошки", 0, "мошка")
	assert.True(t, ok, "мошки is predicted from кошки (shard 0): %+v", d.Parse("мошки"))

	var words []string
	for _, f := range d.Forms(r) {
		words = append(words, f.Word)
		assert.True(t, f.Predicted)
		assert.Equal(t, 1, f.Shard)
	}
	assert.Equal(t, []string{"бревно", "бревна"}, words, "Forms of a shard-1 predicted reading")
}

func TestParseSkipsPredictionForMissingShard(t *testing.T) {
	d := shardedPredictionDict(t)
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], 1) // count
	binary.BigEndian.PutUint16(v[2:4], 0) // para
	binary.BigEndian.PutUint16(v[4:6], 1) // form
	binary.BigEndian.PutUint16(v[6:8], 5) // shard 5 does not exist
	pred, err := internal.BuildDAWGWithValuesBytes([]string{"на"}, [][]byte{v})
	require.NoError(t, err)
	d.d.Prediction = []*internal.DAWG{pred}

	assert.Empty(t, d.Parse("бревна"))
}

func TestShardedPredictionSaveOpen(t *testing.T) {
	d := shardedPredictionDict(t)
	path := filepath.Join(t.TempDir(), "sharded.dat")
	require.NoError(t, d.SaveTo(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(data)
	require.NoError(t, err)
	_, _, err = cont.Section("pred-sharded-0")
	require.NoError(t, err)
	_, _, err = cont.Section("prediction-0")
	require.Error(t, err, "gomorphy 1.2.x reads only prediction-N and must find none")

	fromFile, err := Open(path)
	require.NoError(t, err)
	defer func() { _ = fromFile.Close() }()
	fromBytes, err := OpenBytes(data)
	require.NoError(t, err)
	for name, o := range map[string]*Dictionary{"Open": fromFile, "OpenBytes": fromBytes} {
		assert.True(t, o.d.PredictionSharded, name)
		assert.Equal(t, d.Parse("бревна"), o.Parse("бревна"), name)
		assert.Equal(t, d.Parse("мошки"), o.Parse("мошки"), name)
	}
}

func TestSingleShardPredictionKeepsOldSection(t *testing.T) {
	b := NewBuilder(BuilderOptions{})
	require.NoError(t, b.AddForm("кот", "кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))
	d, err := b.Build()
	require.NoError(t, err)
	require.False(t, d.d.PredictionSharded)

	path := filepath.Join(t.TempDir(), "single.dat")
	require.NoError(t, d.SaveTo(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(data)
	require.NoError(t, err)
	_, _, err = cont.Section("prediction-0")
	require.NoError(t, err)
	_, _, err = cont.Section("pred-sharded-0")
	require.Error(t, err)

	o, err := OpenBytes(data)
	require.NoError(t, err)
	assert.False(t, o.d.PredictionSharded)
	assert.Equal(t, d.Parse("бота"), o.Parse("бота"))
}

func TestOpenRejectsBothPredictionKinds(t *testing.T) {
	d := shardedPredictionDict(t)
	sections, err := d.sections(nil)
	require.NoError(t, err)
	var sharded internal.Section
	for _, s := range sections {
		if s.Name == "pred-sharded-0" {
			sharded = s
		}
	}
	require.NotEmpty(t, sharded.Name)
	sections = append(sections, internal.Section{Name: "prediction-0", Data: sharded.Data, Flags: sharded.Flags})
	path := filepath.Join(t.TempDir(), "both.dat")
	require.NoError(t, internal.SaveContainer(path, sections))

	_, err = Open(path)
	assert.ErrorContains(t, err, "prediction")
}
