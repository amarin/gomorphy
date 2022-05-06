package dag_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/dag"
)

type testIndexStruct struct {
	name     string
	known    []dag.Tag
	wantData string
	wantErr  bool
}

var testCategoryListData = []testIndexStruct{ // nolint:gochecknoglobals
	{"empty_tags_list", []dag.Tag{}, "00", false},
	{"single_empty_tag", []dag.Tag{{dag.EmptyTagName, dag.EmptyTagName}}, "012020202020202020", false},
	{"single_filled_tag", []dag.Tag{{dag.EmptyTagName, "POST"}},
		"0120202020504f5354", false},
	{"couple_of_filled_tags", []dag.Tag{{dag.EmptyTagName, "POST"}, {"POST", "NOUN"}},
		"0220202020504f5354504f53544e4f554e", false},
}

func TestIndex_Idx(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		testName string
		name     dag.TagName
		parent   dag.TagName
		indexed  dag.Idx
		want     dag.TagID
	}{
		{"in_empty", "1111", "", dag.Idx{}, 0},
		{testName: "new_wo_parent", name: "2222", parent: dag.EmptyTagName,
			indexed: dag.Idx{dag.Tag{Parent: "", Name: "1111"}}, want: 1},
		{testName: "new_to_parent", name: "2222", parent: "1111",
			indexed: dag.Idx{
				dag.Tag{Parent: "", Name: "1111"},
			}, want: 1},
		{testName: "existed_to_root", name: "2222",
			indexed: dag.Idx{
				dag.Tag{Parent: dag.EmptyTagName, Name: "1111"},
				dag.Tag{Parent: dag.EmptyTagName, Name: "2222"},
			}, want: 1},
		{testName: "existed_to_parent", name: "3333", parent: "1111",
			indexed: dag.Idx{
				dag.Tag{Parent: dag.EmptyTagName, Name: "1111"},
				dag.Tag{Parent: "1111", Name: "2222"},
			}, want: 2},
	} {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			tt := tt
			x := dag.NewIndex(tt.indexed...)
			require.Equal(t, tt.want, x.Index(tt.name, tt.parent))
		})
	}
}

func TestIndex_ReadFrom(t *testing.T) {
	tests := make([]testIndexStruct, 0)
	tests = append(tests, testCategoryListData...)
	tests = append(tests, []testIndexStruct{
		{ // extra data in buffer is not taken and not an error
			"extra_data_after_error",
			[]dag.Tag{{dag.EmptyTagName, "POST"}},
			"0120202020504f5354FF", false,
		},
		{ // no data len byte should raise
			"err_empty_data",
			nil,
			"",
			true,
		},
		{
			"err_wrong_prefix",
			nil,
			"ff",
			true,
		},
		{ // len of Tag's list greater than available data should raise
			"err_len_mismatch_data",
			[]dag.Tag{{"", "POST"}},
			"02504f535420202020",
			true,
		},
	}...)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			index := dag.NewIndex()
			data, err := hex.DecodeString(tt.wantData)
			require.NoError(t, err)

			_, err = index.BinaryReadFrom(binutils.NewBinaryReader(bytes.NewBuffer(data)))
			require.Equalf(t, tt.wantErr, err != nil, "UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}
			require.Equalf(
				t, len(tt.known), index.Len(),
				"UnmarshalBinary() expected %d items, not %d\nData: %v\nExpect: %v\nGot: %v",
				len(tt.known), index.Len(), tt.wantData, tt.known, index)

			for idx := range tt.known {
				testItem := tt.known[idx]
				indexItem, found := index.Get(dag.TagID(idx))
				require.True(t, found)
				require.Equalf(t, testItem.Name.String(), indexItem.Name.String(),
					"UnmarshalBinary() item %d name mismatch: received %v != %v expected",
					idx, indexItem.Name, testItem.Name)
				require.Equalf(t, testItem.Parent.String(), indexItem.Parent.String(),
					"UnmarshalBinary() item %d parent mismatch: received %v != %v expected in %v",
					idx, indexItem.Parent, testItem.Parent, tt.wantData)
			}
		})
	}
}

func TestIndex_Add(t *testing.T) {
	t.Parallel()

	index := dag.NewIndex()
	post := dag.NewTag("", "POST")
	pos1 := dag.NewTag("", "POS1")

	require.Equal(t, dag.TagID(0), index.Index(post.Name, post.Parent))
	require.Equal(t, dag.TagID(0), index.Index(post.Name, post.Parent)) // duplicated add returns same index
	require.Equal(t, dag.TagID(1), index.Index(pos1.Name, pos1.Parent))
}

func TestIndex_Get(t *testing.T) {
	t.Parallel()

	testIndex := dag.NewIndex(
		*dag.NewTag("", "POST"),
		*dag.NewTag("POST", "NOUN"),
		*dag.NewTag("POST", "VERB"))

	for _, tt := range []struct {
		name  string
		idx   dag.TagID
		found bool
	}{
		{"ok_existed_1", 0, true},
		{"ok_existed_2", 1, true},
		{"ok_existed_3", 2, true},
		{"nok_missed", 3, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			got, found := testIndex.Get(tt.idx)
			require.Equal(t, tt.found, found)
			if found {
				require.Equal(t, got.Name, testIndex[tt.idx].Name)
				require.Equal(t, got.Parent, testIndex[tt.idx].Parent)
			}
		})
	}
}

func TestNewIndex(t *testing.T) {
	POST := dag.NewTag("", "POST")
	NOUN := dag.NewTag("POST", "NOUN")
	VERB := dag.NewTag("POST", "VERB")

	for _, tt := range []struct {
		name string
		tags []dag.Tag
	}{
		{"ok_empty", []dag.Tag{}},
		{"ok_single", []dag.Tag{*POST}},
		{"ok_pair", []dag.Tag{*NOUN, *VERB}},
		{"ok_triplet", []dag.Tag{*NOUN, *VERB, *POST}},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			got := dag.NewIndex(tt.tags...)
			require.Equal(t, len(tt.tags), got.Len())

			for _, g := range tt.tags {
				_, found := got.Find(g.Name)
				require.True(t, found) // looking for existed indexed, expected always found
			}
		})
	}
}

func TestIndex_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct {
		testName  string
		idx       []dag.Tag
		name      dag.TagName
		parent    dag.TagName
		wantID    dag.TagID
		wantFound bool
	}{
		{"find_in_empty", make([]dag.Tag, 0), //nolint:gofumpt
			"", "", 0, false},
		{"find_not_existed_name", []dag.Tag{{"", "1111"}},
			"2222", "", 0, false},
		{"find_existed_root", []dag.Tag{{"", "1111"}, {"2222", "1111"}},
			"1111", "", 0, true},
		{"find_existed_with_parent", []dag.Tag{{"", "1111"}, {"1111", "2222"}},
			"2222", "1111", 1, true},
	} {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			tt := tt
			idx := dag.NewIndex(tt.idx...)
			gotID, gotFound := idx.Find(tt.name)
			require.Equalf(t, tt.wantFound, gotFound, "expected found {%v,%v} in %v", tt.parent, tt.name, idx)
			if gotFound {
				require.Equal(t, tt.wantID, gotID)
			}
		})
	}
}
