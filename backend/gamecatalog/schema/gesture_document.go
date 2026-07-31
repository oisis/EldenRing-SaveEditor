package schema

type GestureData struct {
	GoodsSourceRowID Fact[uint32]        `json:"goodsSourceRowID"`
	IconID           Fact[uint32]        `json:"iconID"`
	Slots            []GestureSlotRecord `json:"slots"`
}

type GestureSlotRecord struct {
	SlotID        Fact[uint32]      `json:"slotID"`
	ItemID        Fact[uint32]      `json:"itemID"`
	Name          Fact[string]      `json:"name"`
	Category      Fact[string]      `json:"category"`
	Flags         Fact[[]string]    `json:"flags"`
	SourceRecords []ParameterRecord `json:"sourceRecords"`
}
