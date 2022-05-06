package opencorpora

// Link defined lemma's link data structure.
// Provides possibilities to switch from one form to another one's.
type Link struct {
	IdAttr   uint32 `xml:"id,attr"`
	FromAttr uint32 `xml:"from,attr"`
	ToAttr   uint32 `xml:"to,attr"`
	TypeAttr uint32 `xml:"type,attr"`
}
