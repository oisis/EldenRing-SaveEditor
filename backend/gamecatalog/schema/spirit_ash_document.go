package schema

type SpiritAshData struct {
	SourceRowID         Fact[uint32] `json:"sourceRowID"`
	IconID              Fact[uint32] `json:"iconID"`
	ReinforceGoodsID    Fact[int32]  `json:"reinforceGoodsID"`
	ReinforceMaterialID Fact[int32]  `json:"reinforceMaterialID"`
	ReinforcePrice      Fact[uint32] `json:"reinforcePrice"`
}
