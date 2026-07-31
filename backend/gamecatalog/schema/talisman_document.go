package schema

type TalismanData struct {
	SourceRowID Fact[uint32]  `json:"sourceRowID"`
	IconID      Fact[uint32]  `json:"iconID"`
	SortID      Fact[uint32]  `json:"sortID"`
	SortGroupID Fact[uint8]   `json:"sortGroupID"`
	Weight      Fact[float64] `json:"weight"`
}
