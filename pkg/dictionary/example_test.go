package dictionary_test

import (
	"fmt"
	"log"
	"os"

	"github.com/amarin/gomorphy/pkg/dictionary"
)

// Example builds a tiny dictionary, saves it, re-opens it and looks a word up.
func Example() {
	b := dictionary.NewBuilder()

	id, err := b.AddLemma("кот", "NOUN", "anim", "masc")
	if err != nil {
		log.Fatal(err)
	}

	if err := b.AddForm(id, "кота", "NOUN", "anim", "masc", "gent"); err != nil {
		log.Fatal(err)
	}

	path := os.TempDir() + "/gomorphy_example.dict"
	if err := b.Compile().SaveTo(path); err != nil {
		log.Fatal(err)
	}

	d, err := dictionary.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = d.Close() }()

	forms, err := d.Lookup("кота")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range forms {
		fmt.Printf("%s %s -> lemma #%d\n", f.Text, f.Ancode, f.LemmaID)
	}

	if err := os.Remove(path); err != nil {
		log.Fatal(err)
	}
	// Output: кота NOUN,anim,masc,gent -> lemma #0
}
