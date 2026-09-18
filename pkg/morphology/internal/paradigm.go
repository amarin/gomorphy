package internal

import "fmt"

// Paradigm is an inflection/conjugation template. A flat uint16 array:
//
//	[suffix_0..suffix_N-1 | tag_0..tag_N-1 | prefix_0..prefix_N-1]
//
// Form i of a paradigm is described by the triple (Suffix(i), Tag(i),
// Prefix(i)).
type Paradigm struct {
	data []uint16
}

// NewParadigm assembles a Paradigm from three equal-length parts.
func NewParadigm(suffixes, tags, prefixes []uint16) Paradigm {
	if len(suffixes) != len(tags) || len(tags) != len(prefixes) {
		panic("internal: paradigm parts must have equal length")
	}
	data := make([]uint16, 0, len(suffixes)*3)
	data = append(data, suffixes...)
	data = append(data, tags...)
	data = append(data, prefixes...)
	return Paradigm{data: data}
}

// NewParadigmFromData creates a Paradigm from flat data in pymorphy2's
// paradigms.array format: [N suffixes | N tags | N prefixes]. The length
// must be divisible by 3. The data is not copied.
func NewParadigmFromData(data []uint16) (Paradigm, error) {
	if len(data)%3 != 0 {
		return Paradigm{}, fmt.Errorf("internal: paradigm length %d is not divisible by 3", len(data))
	}
	return Paradigm{data: data}, nil
}

// Len is the number of forms in the paradigm.
func (p Paradigm) Len() int {
	return len(p.data) / 3
}

// Suffix is form i's suffix id.
func (p Paradigm) Suffix(i int) uint16 {
	return p.data[i]
}

// Tag is form i's tag id.
func (p Paradigm) Tag(i int) uint16 {
	return p.data[p.Len()+i]
}

// Prefix is form i's prefix id.
func (p Paradigm) Prefix(i int) uint16 {
	return p.data[2*p.Len()+i]
}

// Data returns the paradigm's flat data (for serialization).
func (p Paradigm) Data() []uint16 {
	return p.data
}
