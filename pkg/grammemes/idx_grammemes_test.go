package grammemes_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/gomorphy/pkg/grammemes"
	"github.com/stretchr/testify/require"
)

type testIndexStruct struct {
	name     string
	known    []grammemes.Grammeme
	wantData string
	wantErr  bool
}

var testCategoryListData = []testIndexStruct{ // nolint:gochecknoglobals
	{"empty_grammemes_list", []grammemes.Grammeme{}, "00", false},
	{"single_empty_grammeme", []grammemes.Grammeme{{grammemes.Empty, grammemes.Empty}}, "012020202020202020", false},
	{"single_filled_grammeme", []grammemes.Grammeme{{grammemes.Empty, "POST"}},
		"01504f535420202020", false},
	{"couple_of_filled_grammemes", []grammemes.Grammeme{{grammemes.Empty, "POST"}, {"POST", "NOUN"}},
		"02504f5354202020204e4f554e504f5354", false},
}

func TestIndex_Idx(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		testName string
		name     grammemes.Name
		parent   grammemes.Name
		indexed  grammemes.Idx
		want     uint8
	}{
		{"in_empty", "1111", "", grammemes.Idx{}, 0},
		{"new_wo_parent", "2222", "",
			grammemes.Idx{{"", "1111"}}, 1},
		{"new_to_parent", "2222", "1111",
			grammemes.Idx{{"", "1111"}}, 1},
		{"existed_to_root", "2222", "",
			grammemes.Idx{{grammemes.Empty, "1111"}, {grammemes.Empty, "2222"}}, 1},
		{"existed_to_parent", "3333", "1111",
			grammemes.Idx{{grammemes.Empty, "1111"}, {"1111", "2222"}}, 2},
	} {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			tt := tt
			x := grammemes.NewIndex(tt.indexed...)
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
			[]grammemes.Grammeme{{grammemes.Empty, "POST"}},
			"01504f535420202020FF", false,
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
		{ // len of Grammemes list greater than available data should raise
			"err_len_mismatch_data",
			[]grammemes.Grammeme{{"", "POST"}},
			"02504f535420202020",
			true,
		},
	}...)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			index := grammemes.NewIndex()
			data, err := hex.DecodeString(tt.wantData)
			require.NoError(t, err)

			_, err = index.ReadFrom(bytes.NewBuffer(data))
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
				indexItem, found := index.Get(uint8(idx))
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
	index := grammemes.NewIndex()
	post := grammemes.NewGrammeme("", "POST")
	pos1 := grammemes.NewGrammeme("", "POS1")

	require.Equal(t, uint8(0), index.Index(post.Name, post.Parent))
	require.Equal(t, uint8(0), index.Index(post.Name, post.Parent)) // duplicated add returns same index
	require.Equal(t, uint8(1), index.Index(pos1.Name, pos1.Parent))
}

func TestIndex_Get(t *testing.T) {
	testIndex := grammemes.NewIndex(
		*grammemes.NewGrammeme("", "POST"),
		*grammemes.NewGrammeme("POST", "NOUN"),
		*grammemes.NewGrammeme("POST", "ASJF"))

	for _, tt := range []struct {
		name  string
		idx   uint8
		found bool
	}{
		{"ok_existed_1", 0, true},
		{"ok_existed_2", 1, true},
		{"ok_existed_3", 2, true},
		{"nok_missed", 3, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
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
	POST := grammemes.NewGrammeme("", "POST")
	NOUN := grammemes.NewGrammeme("POST", "NOUN")
	ADJF := grammemes.NewGrammeme("POST", "ASJF")

	for _, tt := range []struct {
		name      string
		grammemes []grammemes.Grammeme
	}{
		{"ok_empty", []grammemes.Grammeme{}},
		{"ok_single", []grammemes.Grammeme{*POST}},
		{"ok_pair", []grammemes.Grammeme{*NOUN, *ADJF}},
		{"ok_triplet", []grammemes.Grammeme{*NOUN, *ADJF, *POST}},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			got := grammemes.NewIndex(tt.grammemes...)
			require.Equal(t, len(tt.grammemes), got.Len())

			for _, g := range tt.grammemes {
				_, found := got.Find(g.Name, g.Parent)
				require.True(t, found) // looking for existed indexed, expected always found
			}
		})
	}
}

func TestGrammemeIdx_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct {
		testName     string
		grammemesIdx []grammemes.Grammeme
		name         grammemes.Name
		parent       grammemes.Name
		wantID       uint8
		wantFound    bool
	}{
		{"find_in_empty", make([]grammemes.Grammeme, 0),
			"", "", 0, false},
		{"find_not_existed_name", []grammemes.Grammeme{{"", "1111"}},
			"2222", "", 0, false},
		{"find_not_existed_parent_mismatch", []grammemes.Grammeme{{"", "1111"}},
			"1111", "0000", 0, false},
		{"find_existed_root", []grammemes.Grammeme{{"", "1111"}, {"2222", "1111"}},
			"1111", "", 0, true},
		{"find_existed_with_parent", []grammemes.Grammeme{{"", "1111"}, {"1111", "2222"}},
			"2222", "1111", 1, true},
	} {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			tt := tt
			idx := grammemes.NewIndex(tt.grammemesIdx...)
			gotID, gotFound := idx.Find(tt.name, tt.parent)
			require.Equalf(t, tt.wantFound, gotFound, "expected found {%v,%v} in %v", tt.parent, tt.name, idx)
			if gotFound {
				require.Equal(t, tt.wantID, gotID)
			}
		})
	}
}
