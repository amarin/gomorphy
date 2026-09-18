package opencorpora

// ShardingStrategy decides, while lemmas are processed in corpus order,
// when the current shard should close and a new one start. Each shard's
// suffix strings get their own independent id space (starts at 0,
// addressed as uint16), so no shard may hold more than limit unique
// suffixes.
//
// v1 ships exactly one implementation, FillOnDemand — see
// docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md for why
// (near-zero overhead at today's real N=2 shard count; a second
// strategy, e.g. paradigm-grouped, can be added later as another type
// satisfying this interface without touching ImportFromXML's structure).
type ShardingStrategy interface {
	// Boundary reports whether the CURRENT shard should close before a
	// lemma that would add newSuffixes new, not-yet-seen suffix strings
	// to it is placed — given the shard already holds currentCount
	// unique suffixes and no shard may exceed limit.
	Boundary(currentCount, newSuffixes, limit int) bool
}

// FillOnDemand packs lemmas into the current shard until adding the next
// lemma's new suffixes would exceed limit, then starts a new shard. A
// shard is never closed while still empty, so a single lemma needing
// more than limit suffixes on its own still gets a shard to itself
// rather than looping forever.
type FillOnDemand struct{}

func (FillOnDemand) Boundary(currentCount, newSuffixes, limit int) bool {
	return currentCount > 0 && currentCount+newSuffixes > limit
}
