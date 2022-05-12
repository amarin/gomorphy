package opencorpora_test

import (
	"testing"

	"git.media-tel.ru/railgo/logging"
	"git.media-tel.ru/railgo/logging/zap"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestOpenCorporaLoader_LoadLemmata(t *testing.T) {
	// loader := NewLoader("")
	// _, err := loader.Lemmata()
	//
	// if err != nil {
	// 	t.Fatalf("cant load lemmata: %v", err)
	// }
}

func TestOpenCorporaLoader_Lemmata_SearchWord(t *testing.T) {
	// loader := NewLoader("")
	//
	// lemmata, err := loader.Lemmata()
	//
	// if err != nil {
	// 	t.Fatalf("cant load lemmata: %v", err)
	// }
	//
	// for _, word := range []string{"ёж", "ёлка", "капуста", "капюшон"} {
	// 	if len(lemmata.SearchForms(text.RussianText(word))) == 0 {
	// 		t.Fatalf("Missed word `%v`", word)
	// 	}
	// }
}

func Test_CompileGrammemes(t *testing.T) {
	// grammemesToSearch := []grammeme2.TagName{"POST", "NOUN", "ANim"}
	// loader := new(Loader)
	//
	// if err := loader.CompileGrammemes(); err != nil {
	// 	t.Fatalf("%v", err)
	// } else if grammemesIndex, err := loader.LoadGrammemes(); err != nil {
	// 	t.Fatalf("cant load compiled lemmata: %v", err)
	// } else {
	// 	for _, word := range grammemesToSearch {
	// 		if grammeme, err := grammemesIndex.ByName(word); err != nil {
	// 			t.Fatalf("Missed grammeme `%v`", word)
	// 		} else if _, err := grammemesIndex.Idx(grammeme.Name); err != nil {
	// 			t.Fatalf(
	// 				"Cant take known grammeme `%v` id: %v",
	// 				grammeme.Name, err)
	// 		}
	// 	}
	// }
}

func TestLoader_DownloadUpdate(t *testing.T) {
	logging.MustInit(*logging.CurrentConfig(), new(zap.Backend))
	loader := NewLoader("")
	if _, err := loader.DownloadUpdate(); err != nil {
		t.Errorf("DownloadUpdate() error = %v", err)
	}
}

func TestLoader_UnpackUpdate(t *testing.T) {
	logging.MustInit(*logging.CurrentConfig(), new(zap.Backend))
	loader := NewLoader("")
	if err := loader.UnpackUpdate(); (err != nil) != false {
		t.Errorf("UnpackUpdate() error = %v, wantErr %v", err, false)
	}
}
