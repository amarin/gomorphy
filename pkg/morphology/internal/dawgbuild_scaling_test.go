//go:build scaling

package internal

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestDAWGBuildScales проверяет, что время сборки растёт сублинейно-
// квадратично (не квадратично) с числом ключей — то самое свойство, ради
// которого заменён placement-алгоритм. Требует build tag "scaling" и не
// входит в `make test`/быстрый прогон CI (как integration-тесты — см.
// `-tags integration` в Makefile): запускается явно
// (`go test -tags scaling -run TestDAWGBuildScales -v -timeout 30m ./...`),
// т.к. на верхней границе (5М) занимает несколько минут и несколько
// гигабайт RSS даже при некатастрофическом (не O(n^2)) масштабировании.
func TestDAWGBuildScales(t *testing.T) {

	sizes := []int{100_000, 500_000, 1_000_000, 2_000_000, 5_000_000}
	var perKey []float64

	for _, n := range sizes {
		keys, values := randomWordformKeys(n, 12345)

		start := time.Now()
		d, err := BuildDAWGWithValues(keys, values)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("N=%d: BuildDAWGWithValues: %v", n, err)
		}
		if d == nil {
			t.Fatalf("N=%d: nil DAWG", n)
		}

		perN := elapsed.Seconds() / float64(n)
		perKey = append(perKey, perN)
		t.Logf("N=%9d  elapsed=%-12s  per-key=%.3fµs  dic-len=%d",
			n, elapsed, perN*1e6, len(d.dict))
	}

	// Квадратичный рост (старый findBaseBitset) дал бы РЕЗКИЙ рост per-key:
	// O(n^2) от N=100K до N=5M (в 50 раз) дал бы >2500x, на порядки хуже
	// наблюдаемого. С free-list рост сублинейно-квадратичный (~n^1.3-1.4),
	// не идеально плоский: этот генератор ключей — почти случайные суффиксы
	// + уникальный payload на каждый ключ, т.е. почти без разделяемых
	// суффиксов, в отличие от реальных словоформ (тысячи слов оканчиваются
	// на "-ами", "-ов" и т.п.). Из-за этого dic-массив относительно N сильно
	// раздут (уже ~4.9M слотов на 100K ключей) и быстро уходит в диапазон,
	// где extended-offset поле единицы словаря (см. encodable()) принимает
	// только ~1 из 256 кандидатов — это свойство самого формата dawgdic, не
	// дефект аллокатора. Порог ниже — заведомо ниже, чем дал бы реальный
	// возврат к квадратичному поведению, но выше типичного шума GC/аллокаций.
	const maxAcceptableGrowth = 8.0
	for i := 1; i < len(perKey); i++ {
		ratio := perKey[i] / perKey[0]
		if ratio > maxAcceptableGrowth {
			t.Errorf("per-key cost grew %.1fx from N=%d to N=%d (%.3fµs -> %.3fµs); "+
				"expected sub-quadratic scaling (<%.0fx), got growth consistent with a regression",
				ratio, sizes[0], sizes[i], perKey[0]*1e6, perKey[i]*1e6, maxAcceptableGrowth)
		}
	}
}

// randomWordformKeys генерирует N детерминированных (seed) ключей вида
// "word\x01payload" — с длиной и алфавитом, похожими на реальные
// словоформы (кириллица 4-12 символов) + 4-байтовый payload, как в
// BuildDAWGWithValues.
func randomWordformKeys(n int, seed int64) ([]string, []uint32) {
	rnd := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic test data
	alphabet := []rune("абвгдежзийклмнопрстуфхцчшщъыьэюя")

	keys := make([]string, n)
	values := make([]uint32, n)
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		var w []rune
		length := 4 + rnd.Intn(9) // 4..12
		for j := 0; j < length; j++ {
			w = append(w, alphabet[rnd.Intn(len(alphabet))])
		}
		key := fmt.Sprintf("%s-%d", string(w), i) // suffix guarantees uniqueness
		for seen[key] {
			key = key + "x"
		}
		seen[key] = true
		keys[i] = key
		values[i] = uint32(i)
	}
	return keys, values
}
