//go:build race

package internal

// raceEnabled: the race detector changes allocation counts, so
// AllocsPerRun bounds are only checked without it.
const raceEnabled = true
