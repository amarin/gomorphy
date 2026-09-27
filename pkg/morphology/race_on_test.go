//go:build race

package morphology_test

// raceEnabled: the race detector changes allocation counts, so
// AllocsPerRun bounds are only checked without it.
const raceEnabled = true
