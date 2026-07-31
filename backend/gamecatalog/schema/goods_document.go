package schema

type GoodsData struct {
	SourceRowID    Fact[uint32]   `json:"sourceRowID"`
	IconID         Fact[uint32]   `json:"iconID"`
	SortID         Fact[uint32]   `json:"sortID"`
	SortIDSFV      *Fact[uint32]  `json:"sortID-sfv,omitempty"`
	SortGroupID    Fact[uint8]    `json:"sortGroupID"`
	SortGroupIDSFV *Fact[uint8]   `json:"sortGroupID-sfv,omitempty"`
	GoodsType      Fact[uint16]   `json:"goodsType"`
	Weight         Fact[float64]  `json:"weight"`
	WeightSFV      *Fact[float64] `json:"weight-sfv,omitempty"`
	MaxQuantity    Fact[uint32]   `json:"maxQuantity"`
	MaxRepository  Fact[uint32]   `json:"maxRepository"`
	TutorialFlagID Fact[uint32]   `json:"tutorialFlagID"`
	IsEquipable    Fact[bool]     `json:"isEquipable"`
	IsConsumable   Fact[bool]     `json:"isConsumable"`
	IsDiscardable  Fact[bool]     `json:"isDiscardable"`
	IsDepositable  Fact[bool]     `json:"isDepositable"`
	IsDroppable    Fact[bool]     `json:"isDroppable"`
}
