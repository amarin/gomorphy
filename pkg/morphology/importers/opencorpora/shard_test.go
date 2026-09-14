package opencorpora

import "testing"

func TestFillOnDemandBoundary(t *testing.T) {
	var s FillOnDemand
	cases := []struct {
		name                             string
		currentCount, newSuffixes, limit int
		want                             bool
	}{
		{"empty shard never closes", 0, 100, 10, false},
		{"fits exactly at limit", 5, 5, 10, false},
		{"one over limit closes", 5, 6, 10, true},
		{"already at limit, any addition closes", 10, 1, 10, true},
		{"zero-addition lemma never closes", 10, 0, 10, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.Boundary(tc.currentCount, tc.newSuffixes, tc.limit)
			if got != tc.want {
				t.Errorf("Boundary(%d, %d, %d) = %v, want %v",
					tc.currentCount, tc.newSuffixes, tc.limit, got, tc.want)
			}
		})
	}
}
