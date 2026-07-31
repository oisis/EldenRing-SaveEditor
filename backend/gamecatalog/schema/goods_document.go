package schema

type GoodsData struct {
	SourceRowID    Fact[uint32]  `json:"sourceRowID"`
	IconID         Fact[uint32]  `json:"iconID"`
	SortID         Fact[uint32]  `json:"sortID"`
	SortGroupID    Fact[uint8]   `json:"sortGroupID"`
	GoodsType      Fact[uint16]  `json:"goodsType"`
	Weight         Fact[float64] `json:"weight"`
	MaxQuantity    Fact[uint32]  `json:"maxQuantity"`
	MaxRepository  Fact[uint32]  `json:"maxRepository"`
	TutorialFlagID Fact[uint32]  `json:"tutorialFlagID"`
	IsEquipable    Fact[bool]    `json:"isEquipable"`
	IsConsumable   Fact[bool]    `json:"isConsumable"`
	IsDiscardable  Fact[bool]    `json:"isDiscardable"`
	IsDepositable  Fact[bool]    `json:"isDepositable"`
	IsDroppable    Fact[bool]    `json:"isDroppable"`
}
