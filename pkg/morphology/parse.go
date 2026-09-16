package morphology

import (
	"encoding/binary"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Reading — один разбор словоформы.
type Reading struct {
	Word   string  // словоформа как в словаре (с «ё»)
	Normal string  // начальная форма (лемма)
	Tag    string  // граммемный тег, например "NOUN,anim,masc,sing,nomn"
	Para   uint16  // id парадигмы — уникален только вместе с Shard
	Form   uint16  // индекс формы в парадигме
	Shard  int     // индекс шарда словаря; всегда 0 для нешардированных словарей
	Dict   int     // индекс словаря в MultiDictionary; всегда 0 для Dictionary.Parse напрямую
	Prob   float64 // вероятность разбора (0, если probability недоступен)
}

// Parse разбирает слово и возвращает все чтения словаря, отсортированные
// по вероятности (убыванию). Для слов вне словаря пытается предсказать
// чтения по prediction-DAWG (окончания). Возвращает nil, если разборы
// не найдены. Вход приводится к нижнему регистру.
func (x *Dictionary) Parse(word string) []Reading {
	if x == nil || x.d == nil || len(x.d.Words) == 0 {
		return nil
	}
	word = strings.ToLower(word)

	if readings := x.exact(word); len(readings) > 0 {
		return readings
	}
	return x.predict(word)
}

// shardExactResult — результат exactInShard для одного шарда.
type shardExactResult struct {
	readings []Reading
	hasProb  bool
}

// exact собирает чтения слова, найденного в словаре, с учётом подмен
// CharPolicy (е→ё) и сортирует по вероятности. Шарды опрашиваются
// параллельно (по горутине на шард), результаты склеиваются.
func (x *Dictionary) exact(word string) []Reading {
	results := make([]shardExactResult, len(x.d.Words))

	var wg sync.WaitGroup
	for shard, dawg := range x.d.Words {
		wg.Add(1)
		go func(shard int, dawg *internal.DAWG) {
			defer wg.Done()
			results[shard] = x.exactInShard(shard, dawg, word)
		}(shard, dawg)
	}
	wg.Wait()

	var readings []Reading
	hasProb := false
	for _, r := range results {
		readings = append(readings, r.readings...)
		hasProb = hasProb || r.hasProb
	}

	if hasProb {
		sort.SliceStable(readings, func(i, j int) bool {
			return readings[i].Prob > readings[j].Prob
		})
	}
	return readings
}

// exactInShard собирает чтения слова из одного шарда. Вызывается
// параллельно с другими шардами из exact — только чтение, общего
// изменяемого состояния между горутинами нет.
func (x *Dictionary) exactInShard(shard int, dawg *internal.DAWG, word string) shardExactResult {
	items := dawg.SimilarItems(word, x.d.CharPolicy, x.d.Alphabet)
	if len(items) == 0 {
		return shardExactResult{}
	}

	var res shardExactResult
	res.readings = make([]Reading, 0, len(items))
	for _, it := range items {
		for _, v := range it.Values {
			r, ok := x.reading(shard, it.Key, v)
			if !ok {
				continue
			}
			if x.d.Probability != nil {
				r.Prob = float64(x.d.Probability.Find(it.Key+":"+r.Tag)) / 1e6
				if r.Prob > 0 {
					res.hasProb = true
				}
			}
			res.readings = append(res.readings, r)
		}
	}
	return res
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
		readings = append(readings, x.predictForPrefix(id, splits, seen)...)
	}
	return readings
}

// predictForPrefix predicts readings against a single prefix's
// prediction-DAWG (x.d.Prediction[id]), widening from the longest suffix
// split (splits[len(splits)-1]) toward shorter ones until at least 2 total
// matches accumulate — pymorphy2's KnownSuffixAnalyzer heuristic: trust a
// long, specific suffix match over a short, common one when it exists.
// seen dedups (word, lemma, tag) triples across all prefixes tried by the
// caller and is mutated in place.
//
// Predictions always resolve against shard 0: the prediction-DAWG feature
// currently exists only for pymorphy2 imports, which are never sharded
// (see docs/superpowers/specs/2026-09-14-suffix-sharding-design.md).
func (x *Dictionary) predictForPrefix(id int, splits [][2]string, seen map[string]bool) []Reading {
	const predictionShard = 0

	var readings []Reading
	totalCount := 0

	for i := len(splits) - 1; i >= 0; i-- {
		wordStart, wordEnd := splits[i][0], splits[i][1]
		// Prediction DAWGs are never recompiled under Dictionary.Alphabet
		// (out of scope — see docs/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md's
		// non-goals): always nil here, even for a dictionary whose Words
		// DAWG uses a dense alphabet.
		for _, it := range x.d.Prediction[id].SimilarItems(wordEnd, x.d.CharPolicy, nil) {
			for _, v := range it.Values {
				if len(v) < 6 {
					continue
				}
				count := int(binary.BigEndian.Uint16(v[:2]))
				paraNum := binary.BigEndian.Uint16(v[2:4])
				form := binary.BigEndian.Uint16(v[4:6])

				para, ok := x.paradigm(predictionShard, paraNum)
				if !ok || form >= uint16(para.Len()) {
					continue
				}
				if !productive(x.paradigmTag(para, int(form))) {
					continue
				}
				totalCount += count

				r := x.readingForm(predictionShard, wordStart+it.Key, paraNum, form)
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
	return readings
}

// reading декодирует payload-запись words.dawg (4 байта BE: para, form) в
// указанном шарде.
func (x *Dictionary) reading(shard int, word string, value []byte) (Reading, bool) {
	if len(value) < 4 {
		return Reading{}, false
	}
	para := binary.BigEndian.Uint16(value[:2])
	form := binary.BigEndian.Uint16(value[2:4])
	return x.readingForm(shard, word, para, form), true
}

// readingForm строит Reading по парадигме и форме в указанном шарде
// (норма = prefix₀ + stem + suffix₀ для form≠0, иначе — само слово).
func (x *Dictionary) readingForm(shard int, word string, paraNum, form uint16) Reading {
	para, ok := x.paradigm(shard, paraNum)
	if !ok || int(form) >= para.Len() {
		return Reading{Word: word, Para: paraNum, Form: form, Shard: shard}
	}

	prefix, suffix := x.paradigmAffix(shard, para, int(form))
	norm := word
	if form != 0 {
		stem := strings.TrimPrefix(word, prefix)
		stem = strings.TrimSuffix(stem, suffix)
		p0, s0 := x.paradigmAffix(shard, para, 0)
		norm = p0 + stem + s0
	}

	return Reading{
		Word:   word,
		Normal: norm,
		Tag:    x.paradigmTag(para, int(form)),
		Para:   paraNum,
		Form:   form,
		Shard:  shard,
	}
}

func (x *Dictionary) paradigm(shard int, id uint16) (internal.Paradigm, bool) {
	if shard < 0 || shard >= len(x.d.Paradigms) {
		return internal.Paradigm{}, false
	}
	if int(id) < len(x.d.Paradigms[shard]) {
		return x.d.Paradigms[shard][id], true
	}
	return internal.Paradigm{}, false
}

// paradigmAffix возвращает префикс и суффикс формы парадигмы для
// указанного шарда (пустые при выходе за границы). Prefixes общий для
// всех шардов; Suffixes — свой на шард.
func (x *Dictionary) paradigmAffix(shard int, para internal.Paradigm, form int) (prefix, suffix string) {
	if form >= para.Len() {
		return "", ""
	}
	var suffixes []string
	if shard >= 0 && shard < len(x.d.Suffixes) {
		suffixes = x.d.Suffixes[shard]
	}
	return strAt(x.d.Prefixes, para.Prefix(form)), strAt(suffixes, para.Suffix(form))
}

// paradigmTag возвращает имя тега формы парадигмы. TagSet общий для всех
// шардов, поэтому шард не нужен.
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

// productive — граммема не входит в nonproductiveGrammemes.
func productive(tag string) bool {
	if tag == "" {
		return false
	}
	for part := range strings.SplitSeq(tag, ",") {
		if slices.Contains(nonproductiveGrammemes, part) {
			return false
		}
	}
	return true
}

// nonproductiveGrammemes — граммемы, для которых предсказание не даёт
// продуктивных разборов (pymorphy2).
var nonproductiveGrammemes = []string{"NUMR", "NPRO", "PRED", "PREP", "CONJ", "PRCL", "INTJ", "Apro"}

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
