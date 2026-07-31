package schema

type TalismanData struct {
	SourceRowID    Fact[uint32]   `json:"sourceRowID"`
	IconID         Fact[uint32]   `json:"iconID"`
	SortID         Fact[uint32]   `json:"sortID"`
	SortIDSFV      *Fact[uint32]  `json:"sortID-sfv,omitempty"`
	SortGroupID    Fact[uint8]    `json:"sortGroupID"`
	SortGroupIDSFV *Fact[uint8]   `json:"sortGroupID-sfv,omitempty"`
	Weight         Fact[float64]  `json:"weight"`
	WeightSFV      *Fact[float64] `json:"weight-sfv,omitempty"`
}
