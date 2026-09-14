package morphology

import (
	"encoding/binary"
	"sort"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Reading — один разбор словоформы.
type Reading struct {
	Word   string  // словоформа как в словаре (с «ё»)
	Normal string  // начальная форма (лемма)
	Tag    string  // граммемный тег, например "NOUN,anim,masc,sing,nomn"
	Para   uint16  // id парадигмы
	Form   uint16  // индекс формы в парадигме
	Prob   float64 // вероятность разбора (0, если probability недоступен)
}

// Parse разбирает слово и возвращает все чтения словаря, отсортированные
// по вероятности (убыванию). Для слов вне словаря пытается предсказать
// чтения по prediction-DAWG (окончания). Возвращает nil, если разборы
// не найдены. Вход приводится к нижнему регистру.
func (x *Dictionary) Parse(word string) []Reading {
	if x == nil || x.d == nil || x.d.Words == nil {
		return nil
	}
	word = strings.ToLower(word)

	if readings := x.exact(word); len(readings) > 0 {
		return readings
	}
	return x.predict(word)
}

// exact собирает чтения слова, найденного в словаре, с учётом подмен
// CharPolicy (е→ё) и сортирует по вероятности.
func (x *Dictionary) exact(word string) []Reading {
	items := x.d.Words.SimilarItems(word, x.d.CharPolicy)
	if len(items) == 0 {
		return nil
	}

	readings := make([]Reading, 0, len(items))
	hasProb := false
	for _, it := range items {
		for _, v := range it.Values {
			r, ok := x.reading(it.Key, v)
			if !ok {
				continue
			}
			if x.d.Probability != nil {
				r.Prob = float64(x.d.Probability.Find(it.Key+":"+r.Tag)) / 1e6
				if r.Prob > 0 {
					hasProb = true
				}
			}
			readings = append(readings, r)
		}
	}

	if hasProb {
		sort.SliceStable(readings, func(i, j int) bool {
			return readings[i].Prob > readings[j].Prob
		})
	}
	return readings
}

// predict ищет чтения для несловарного слова по окончаниям в prediction-DAWG
// (алгоритм KnownSuffixAnalyzer из pymorphy2, как в opennota/morph).
func (x *Dictionary) predict(word string) []Reading {
	if len(x.d.Prediction) == 0 {
		return nil
	}
	splits, ok := suffixSplits(word, 5)
	if !ok {
		return nil
	}

	var readings []Reading
	seen := make(map[string]bool)

	for id, pref := range x.d.Prefixes {
		if id >= len(x.d.Prediction) || x.d.Prediction[id] == nil {
			continue
		}
		if !strings.HasPrefix(word, pref) {
			continue
		}

		totalCount := 0
		for i := len(splits) - 1; i >= 0; i-- {
			wordStart, wordEnd := splits[i][0], splits[i][1]
			for _, it := range x.d.Prediction[id].SimilarItems(wordEnd, x.d.CharPolicy) {
				for _, v := range it.Values {
					if len(v) < 6 {
						continue
					}
					count := int(binary.BigEndian.Uint16(v[:2]))
					paraNum := binary.BigEndian.Uint16(v[2:4])
					form := binary.BigEndian.Uint16(v[4:6])

					para, ok := x.paradigm(paraNum)
					if !ok || form >= uint16(para.Len()) {
						continue
					}
					if !productive(x.paradigmTag(para, int(form))) {
						continue
					}
					totalCount += count

					r := x.readingForm(wordStart+it.Key, paraNum, form)
					key := r.Word + "\x00" + r.Normal + "\x00" + r.Tag
					if seen[key] {
						continue
					}
					seen[key] = true
					readings = append(readings, r)
				}
			}
			if totalCount > 1 {
				break
			}
		}
	}
	return readings
}

// reading декодирует payload-запись words.dawg (4 байта BE: para, form).
func (x *Dictionary) reading(word string, value []byte) (Reading, bool) {
	if len(value) < 4 {
		return Reading{}, false
	}
	para := binary.BigEndian.Uint16(value[:2])
	form := binary.BigEndian.Uint16(value[2:4])
	return x.readingForm(word, para, form), true
}

// readingForm строит Reading по парадигме и форме (норма = prefix₀ + stem + suffix₀
// для form≠0, иначе — само слово).
func (x *Dictionary) readingForm(word string, paraNum, form uint16) Reading {
	para, ok := x.paradigm(paraNum)
	if !ok || int(form) >= para.Len() {
		return Reading{Word: word, Para: paraNum, Form: form}
	}

	prefix, suffix := x.paradigmAffix(para, int(form))
	norm := word
	if form != 0 {
		stem := strings.TrimPrefix(word, prefix)
		stem = strings.TrimSuffix(stem, suffix)
		p0, s0 := x.paradigmAffix(para, 0)
		norm = p0 + stem + s0
	}

	return Reading{
		Word:   word,
		Normal: norm,
		Tag:    x.paradigmTag(para, int(form)),
		Para:   paraNum,
		Form:   form,
	}
}

func (x *Dictionary) paradigm(id uint16) (internal.Paradigm, bool) {
	if int(id) < len(x.d.Paradigms) {
		return x.d.Paradigms[id], true
	}
	return internal.Paradigm{}, false
}

// paradigmAffix возвращает префикс и суффикс формы парадигмы (пустые при
// выходе за границы).
func (x *Dictionary) paradigmAffix(para internal.Paradigm, form int) (prefix, suffix string) {
	if form >= para.Len() {
		return "", ""
	}
	return strAt(x.d.Prefixes, para.Prefix(form)), strAt(x.d.Suffixes, para.Suffix(form))
}

// paradigmTag возвращает имя тега формы парадигмы.
func (x *Dictionary) paradigmTag(para internal.Paradigm, form int) string {
	if form >= para.Len() {
		return ""
	}
	if x.d.TagSet == nil {
		return ""
	}
	return x.d.TagSet.TagName(para.Tag(form))
}

func strAt(ar []string, i uint16) string {
	if int(i) < len(ar) {
		return ar[i]
	}
	return ""
}

// suffixSplits делит слово по рунам на пары (префикс, окончание) для суффиксов
// длиной 1..min(max, len). Возвращает false для пустого слова.
func suffixSplits(word string, max int) ([][2]string, bool) {
	rr := []rune(word)
	n := len(rr)
	if n == 0 {
		return nil, false
	}
	if max > n {
		max = n
	}
	out := make([][2]string, 0, max)
	for i := 1; i <= max; i++ {
		out = append(out, [2]string{string(rr[:n-i]), string(rr[n-i:])})
	}
	return out, true
}

// nonproductiveGrammemes — граммемы, для которых предсказание не даёт
// продуктивных разборов (pymorphy2).
var nonproductiveGrammemes = []string{"NUMR", "NPRO", "PRED", "PREP", "CONJ", "PRCL", "INTJ", "Apro"}

func productive(tag string) bool {
	if tag == "" {
		return false
	}
	for _, g := range nonproductiveGrammemes {
		if strings.Contains(tag, g) {
			return false
		}
	}
	return true
}
