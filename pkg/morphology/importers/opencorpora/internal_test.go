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

func TestStripCmp2Prefix(t *testing.T) {
	cases := []struct {
		name          string
		forms         []formGrams
		wantStemInput []string
		wantPrefixes  []string
		wantOK        bool
	}{
		{
			name: "no Cmp2 forms",
			forms: []formGrams{
				{text: "кот", gramm: "NOUN,anim,masc,sing,nomn"},
				{text: "кота", gramm: "NOUN,anim,masc,sing,gent"},
			},
			wantStemInput: []string{"кот", "кота"},
			wantPrefixes:  []string{"", ""},
			wantOK:        true,
		},
		{
			name: "Cmp2 forms strip по",
			forms: []formGrams{
				{text: "яснее", gramm: "COMP,Qual"},
				{text: "ясней", gramm: "COMP,Qual,V-ej"},
				{text: "пояснее", gramm: "COMP,Qual,Cmp2"},
				{text: "поясней", gramm: "COMP,Qual,Cmp2,V-ej"},
			},
			wantStemInput: []string{"яснее", "ясней", "яснее", "ясней"},
			wantPrefixes:  []string{"", "", "по", "по"},
			wantOK:        true,
		},
		{
			name: "Cmp2 form without по prefix falls back for the whole lemma",
			forms: []formGrams{
				{text: "яснее", gramm: "COMP,Qual"},
				{text: "СЛОМАНО", gramm: "COMP,Qual,Cmp2"},
			},
			wantStemInput: []string{"яснее", "СЛОМАНО"},
			wantPrefixes:  []string{"", ""},
			wantOK:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stemInput, prefixes, ok := stripCmp2Prefix(tc.forms)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if len(stemInput) != len(tc.wantStemInput) {
				t.Fatalf("stemInput = %v, want %v", stemInput, tc.wantStemInput)
			}
			for i := range stemInput {
				if stemInput[i] != tc.wantStemInput[i] {
					t.Errorf("stemInput[%d] = %q, want %q", i, stemInput[i], tc.wantStemInput[i])
				}
			}
			if len(prefixes) != len(tc.wantPrefixes) {
				t.Fatalf("prefixes = %v, want %v", prefixes, tc.wantPrefixes)
			}
			for i := range prefixes {
				if prefixes[i] != tc.wantPrefixes[i] {
					t.Errorf("prefixes[%d] = %q, want %q", i, prefixes[i], tc.wantPrefixes[i])
				}
			}
		})
	}
}

func TestParadigmKeyHashIncludesPrefix(t *testing.T) {
	sk := []uint16{0, 1}
	tk := []uint16{5, 6}
	h1 := paradigmKeyHash([]uint16{0, 0}, sk, tk)
	h2 := paradigmKeyHash([]uint16{0, 1}, sk, tk)
	if h1 == h2 {
		t.Errorf("paradigmKeyHash must differ when prefix IDs differ (same suffix+tag IDs): got equal hashes for %v vs %v", h1, h2)
	}
}
