package morphology

import "testing"

func TestProductive(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		want bool
	}{
		{"empty", "", false},
		{"plain productive", "NOUN,anim,masc,sing,nomn", true},
		{"exact nonproductive token", "NUMR", false},
		{"nonproductive token among others", "NOUN,CONJ,sing", false},
		{"substring only, not a token", "NOUNCONJ,sing", true},
		{"substring inside another token", "XPRCLX,foo", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := productive(tc.tag); got != tc.want {
				t.Errorf("productive(%q) = %v, want %v", tc.tag, got, tc.want)
			}
		})
	}
}
