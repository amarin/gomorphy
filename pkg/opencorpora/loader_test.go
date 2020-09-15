package opencorpora_test

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

type SimpleFormatter struct{}

func (x SimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(strings.ToUpper(entry.Level.String()[:4]) + " " + entry.Message + "\n"), nil
}

func TestOpenCorporaLoader_LoadLemmata(t *testing.T) {
	logrus.SetFormatter(SimpleFormatter{})

	loader := NewLoader(nil, "")
	_, err := loader.Lemmata()

	if err != nil {
		t.Fatalf("cant load lemmata: %v", err)
	}
}

func TestOpenCorporaLoader_Lemmata_SearchWord(t *testing.T) {
	logrus.SetFormatter(SimpleFormatter{})

	loader := NewLoader(nil, "")

	lemmata, err := loader.Lemmata()

	if err != nil {
		t.Fatalf("cant load lemmata: %v", err)
	}

	for _, word := range []string{"ёж", "ёлка", "капуста", "капюшон"} {
		if len(lemmata.SearchForms(text.RussianText(word))) == 0 {
			t.Fatalf("Missed word `%v`", word)
		}
	}
}

func Test_CompileGrammemes(t *testing.T) {
	logrus.SetFormatter(SimpleFormatter{})

	grammemesToSearch := []grammemes.GrammemeName{"POST", "NOUN", "ANim"}
	loader := new(Loader)

	if err := loader.CompileGrammemes(); err != nil {
		t.Fatalf("%v", err)
	} else if grammemesIndex, err := loader.LoadGrammemes(); err != nil {
		t.Fatalf("cant load compiled lemmata: %v", err)
	} else {
		for _, word := range grammemesToSearch {
			if grammeme, err := grammemesIndex.ByName(word); err != nil {
				t.Fatalf("Missed grammeme `%v`", word)
			} else if _, err := grammemesIndex.Idx(grammeme.Name); err != nil {
				t.Fatalf(
					"Cant take known grammeme `%v` id: %v",
					grammeme.Name, err)
			}
		}
	}
}

func TestLoader_DownloadUpdate(t *testing.T) {
	loader := Loader{}
	if _, err := loader.DownloadUpdate(); err != nil {
		t.Errorf("DownloadUpdate() error = %v", err)
	}
}

func TestLoader_UnpackUpdate(t *testing.T) {
	loader := Loader{}
	if err := loader.UnpackUpdate(); (err != nil) != false {
		t.Errorf("UnpackUpdate() error = %v, wantErr %v", err, false)
	}
}
