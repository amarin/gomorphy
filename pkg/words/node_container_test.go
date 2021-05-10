package words_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/grammeme"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/words"
)

func TestNodesContainer_String(t *testing.T) {
	POST := &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammeme.NewIndex(*POST, *NOUN)

	for _, tt := range []struct {
		name   string
		parent *words.Node
		add    []string
		want   string
	}{
		{"empty_no_parent",
			nil,
			[]string{},
			"-->С{}"},
		{"empty_with_parent",
			words.NewMappingNode(nil, 'а'),
			[]string{},
			"N{а}->С{}"},
		{"filled_no_parent",
			nil,
			[]string{"курочка", "ряба"},
			"-->С{кр}"},
		{"filled_with_parent",
			words.NewMappingNode(nil, 'а'),
			[]string{"цветы", "самоцветы"},
			"N{а}->С{сц}"},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			container := words.NewNodeContainer(tt.parent)
			if len(tt.add) != 0 {
				for _, wordText := range tt.add {
					word := words.NewWord(indexA, text.RussianText(wordText), POST, NOUN)
					if _, err := container.AddWord(word); err != nil {
						t.Fatalf("cant add word `%v` to test container %v: %v", word, tt.add, err)
					}
				}
			}

			if got := container.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodesContainer_Len(t *testing.T) {
	POST := &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammeme.NewIndex(*POST, *NOUN)

	for _, tt := range []struct {
		name     string
		addWords []string
		want     int
	}{
		{"empty_len_0", []string{}, 0},
		{"single_len_1", []string{"рондо"}, 1},
		{"couple_len_2", []string{"скоморохи", "цветы"}, 2},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			container := words.NewNodeContainer(nil)
			if len(tt.addWords) != 0 {
				for _, wordText := range tt.addWords {
					word := words.NewWord(indexA, text.RussianText(wordText), POST, NOUN)
					if _, err := container.AddWord(word); err != nil {
						t.Fatalf("cant add word `%v` to test container %v: %v", word, container, err)
					}
				}
			}
			if got := container.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v, children len %v", got, tt.want, len(container.Children()))
			}
		})
	}
}

func TestNodesContainer_FindChild(t *testing.T) {
	container := words.NewNodeContainer(nil)

	for _, char := range strings.Split(words.RussianAlphabetLower, "") {
		r := []rune(char)[0]
		if _, err := container.FindChild(r); err == nil {
			t.Fatalf("No error while search non-existed rune %v.FindChild('%v')", container, r)
		}
	}

	testRune := 'я'

	container.Child(testRune)

	if _, err := container.FindChild(testRune); err != nil {
		t.Fatalf("error while search existed rune %v.FindChild('%v')", container, testRune)
	}
}

func TestNodesContainer_HasChild(t *testing.T) {
	container := words.NewNodeContainer(nil)

	for _, char := range strings.Split(words.RussianAlphabetLower, "") {
		r := []rune(char)[0]
		if container.HasChild(r) {
			t.Fatalf("No error while search non-existed rune %v.HasChild('%v')", container, r)
		}
	}

	testRune := 'я'

	container.Child(testRune)

	if !container.HasChild(testRune) {
		t.Fatalf("error while search existed rune %v.HasChild('%v')", container, testRune)
	}
}

func TestNodesContainer_Child(t *testing.T) {
	container := words.NewNodeContainer(nil)
	child := container.Child('б')

	if child == nil {
		t.Fatalf("%v child nil", container)
	} else if child.Parent() != nil {
		t.Fatalf("%v child parent %p != nil", container, child.Parent())
	}

	c2 := child.Children().Child('ю')
	if c2.Parent() != child {
		t.Fatalf("%v child parent %p != nil", container, c2.Parent())
	}
}

func TestNodesContainer_SearchForms(t *testing.T) {
	var (
		POST        = &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
		NOUN        = &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
		indexA      = grammeme.NewIndex(*POST, *NOUN)
		indexedWord = words.NewWord(indexA, "кот", POST, NOUN)
		missedWord  = words.NewWord(indexA, "котлета", POST, NOUN)
	)

	container := words.NewNodeContainer(nil)
	if _, err := container.AddWord(indexedWord); err != nil {
		t.Fatalf("cant add word to grammemesIndex: %v", err)
	}

	if found := container.SearchForms(missedWord.Text()); len(found) != 0 {
		t.Fatalf("found grammemes for unknown word %v \nin %v", missedWord.Text(), container)
	}

	if found := container.SearchForms(""); len(found) != 0 {
		t.Fatalf("found grammemes for empty word \nin %v", container)
	}

	found := container.SearchForms(indexedWord.Text())

	if len(found) != 1 {
		t.Fatalf("cant find grammemes for existed word %v \nin %v", indexedWord.Text(), container)
	}

	found = container.SearchForms(text.RussianText(strings.ToUpper(string(indexedWord.Text()))))

	if len(found) != 1 {
		t.Fatalf("cant find grammemes for existed word %v in uppercase \nin %v", indexedWord.Text(), container)
	}

	variant := found[0]

	if variant.Len() != 2 {
		t.Fatalf("unexpected grammemes list len %d \nin %v", variant.Len(), variant)
	} else if !variant.EqualTo(indexedWord.Grammemes()) {
		t.Fatalf("unexpected grammemes: \nwant: %v \n got: %v", variant, indexedWord.Grammemes())
	}
}

func BenchmarkNodesContainer_SearchForms(b *testing.B) {
	POST := &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammeme.NewIndex(*POST, *NOUN)
	container := words.NewNodeContainer(nil)

	var benchmarkSearchFormsWith = []text.RussianText{
		"я",
		"мы",
		"она",
		"тебе",
		"плита",
		"сердце",
		"колбаса",
		"спасение",
		"солнечный",
		"опустошение",
		"спиритуализм",
		"гальванизация",
		"взяточничество",
		"геополитический",
		"человеконенавистничество",
	}

	for _, testWord := range benchmarkSearchFormsWith {
		if _, err := container.AddWord(words.NewWord(indexA, testWord, POST, NOUN)); err != nil {
			b.Fatalf("cant prepare test data: %v", err)
		}
	}

	for _, testWord := range benchmarkSearchFormsWith {
		testWord := testWord // pin tt
		b.Run(fmt.Sprintf("search_%d_chars", testWord.Len()), func(b *testing.B) {
			testWord := testWord // pin tt
			for i := 0; i < b.N; i++ {
				_ = container.SearchForms(testWord)
			}
		})
	}
}

func TestNodesContainer_Slice(t *testing.T) {
	POST := &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammeme.NewIndex(*POST, *NOUN)

	testWord := text.RussianText("квалификация")
	container := words.NewNodeContainer(nil)

	if _, err := container.AddWord(words.NewWord(indexA, testWord, POST, NOUN)); err != nil {
		t.Fatalf("cant prepare test data: %v", err)
	}

	sliced := container.Slice()
	if len(sliced) != testWord.Len() {
		t.Fatalf("Expected len %d, not %d: \no: %v\ns: %v", testWord.Len(), len(sliced), container, sliced)
	}
}

//
// func TestNodesContainer_Children(t *testing.T) {
// 	t.Logf("TestNodesContainer_Children not implemented")
// }
//
// func TestNodesContainer_AddWord(t *testing.T) {
// 	t.Logf("TestNodesContainer_AddWord not implemented")
// }
