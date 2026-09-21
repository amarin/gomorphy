package morphology

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mascNomn = "NOUN,anim,masc,sing,nomn"
	mascGent = "NOUN,anim,masc,sing,gent"
	femnNomn = "NOUN,inan,femn,sing,nomn"
	femnGent = "NOUN,inan,femn,sing,gent"
)

func mergeTestDict(t *testing.T, triples ...[3]string) *Dictionary {
	t.Helper()
	b := NewBuilder(BuilderOptions{})
	for _, tr := range triples {
		require.NoError(t, b.AddForm(tr[0], tr[1], tr[2]))
	}
	d, err := b.Build()
	require.NoError(t, err)
	return d
}

func TestMergeDenseOutputAndPrediction(t *testing.T) {
	base := mergeTestDict(t,
		[3]string{"кот", "кот", mascNomn},
		[3]string{"кота", "кот", mascGent},
	)
	overlay := mergeTestDict(t,
		[3]string{"мышь", "мышь", femnNomn},
		[3]string{"мыши", "мышь", femnGent},
	)

	merged, err := Merge(base, []*Dictionary{overlay}, MergeAdd)
	require.NoError(t, err)
	require.NotNil(t, merged)

	require.NotNil(t, merged.d, "merged dictionary must wrap an internal dictionary")
	assert.NotNil(t, merged.d.Alphabet, "Merge output must be dense (Alphabet set)")
	assert.Len(t, merged.d.Prediction, 1, "Merge must rebuild the prediction")
	assert.Equal(t, "merge", merged.d.Info.Source)
	assert.Equal(t, mergeTagSetName, merged.TagSetName())
}

func TestMergeInfoInheritance(t *testing.T) {
	base := NewBuilder(BuilderOptions{Language: "en", Source: "custom"})
	require.NoError(t, base.AddForm("кот", "кот", mascNomn))
	baseDict, err := base.Build()
	require.NoError(t, err)
	baseDict.d.Info.SourceVersion = "v9"
	baseDict.d.Info.Description = "custom base overlay"

	overlay := mergeTestDict(t,
		[3]string{"мышь", "мышь", femnNomn},
		[3]string{"мыши", "мышь", femnGent},
	)

	merged, err := Merge(baseDict, []*Dictionary{overlay}, MergeAdd)
	require.NoError(t, err)

	assert.Equal(t, "merge", merged.d.Info.Source, "Source is stamped by Merge")
	assert.Equal(t, "v9", merged.d.Info.SourceVersion, "SourceVersion is inherited from base Info")
	assert.Equal(t, "custom base overlay", merged.d.Info.Description, "Description is inherited from base Info")
	assert.Equal(t, "en", merged.Language(), "language is inherited from the base dictionary")
	assert.Same(t, baseDict.d.CharPolicy, merged.d.CharPolicy, "CharPolicy is inherited from the base dictionary")
	assert.Equal(t, mergeTagSetName, merged.TagSetName())
}

func TestMergeEmptyErrNoEntries(t *testing.T) {
	empty, err := buildFromEntries(BuilderOptions{}, nil, "empty")
	require.NoError(t, err, "a degenerate empty dictionary is buildable")

	_, err = Merge(empty, nil, MergeAdd)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoEntries), "empty merge must wrap ErrNoEntries, got: %v", err)
}
