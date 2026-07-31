package schema

type SpiritAshData struct {
	SourceRowID         Fact[uint32]  `json:"sourceRowID"`
	IconID              Fact[uint32]  `json:"iconID"`
	SortID              Fact[uint32]  `json:"sortID"`
	SortIDSFV           *Fact[uint32] `json:"sortID-sfv,omitempty"`
	SortGroupID         Fact[uint8]   `json:"sortGroupID"`
	SortGroupIDSFV      *Fact[uint8]  `json:"sortGroupID-sfv,omitempty"`
	ReinforceGoodsID    Fact[int32]   `json:"reinforceGoodsID"`
	ReinforceMaterialID Fact[int32]   `json:"reinforceMaterialID"`
	ReinforcePrice      Fact[uint32]  `json:"reinforcePrice"`
}
