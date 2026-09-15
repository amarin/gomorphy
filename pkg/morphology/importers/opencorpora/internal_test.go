package opencorpora

import (
	"testing"
	"unicode/utf8"
)

func TestLCP(t *testing.T) {
	cases := []struct {
		name  string
		texts []string
		want  string
	}{
		{"empty", nil, ""},
		{"single", []string{"кот"}, "кот"},
		{"identical", []string{"дом", "дом"}, "дом"},
		{"simple shared prefix", []string{"кот", "кота"}, "кот"},
		{"no shared prefix", []string{"кот", "мышь"}, ""},
		{
			// "абажурнее" (а = d0 b0) vs "побажурнее" (п = d0 bf): the
			// first byte of both matches (d0), the second byte differs
			// (b0 vs bf). A naive byte-level LCP would cut at 1 byte,
			// splitting the multi-byte characters "а"/"п" in half. The
			// correct LCP is "" — the first character differs.
			name:  "byte-boundary collision does not produce a mid-rune cut",
			texts: []string{"абажурнее", "побажурнее"},
			want:  "",
		},
		{
			// "поправимее" vs "попоправимее": the shared prefix is the
			// three whole characters "поп" (6 bytes) — the 7th byte
			// starts a new, different character in both texts, so no
			// trimming is needed here; this case guards against an
			// over-eager fix that trims valid boundaries too.
			name:  "shared multi-rune prefix stays intact",
			texts: []string{"поправимее", "попоправимее"},
			want:  "поп",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lcp(tc.texts)
			if got != tc.want {
				t.Errorf("lcp(%v) = %q, want %q", tc.texts, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("lcp(%v) = %q is not valid UTF-8", tc.texts, got)
			}
		})
	}
}
