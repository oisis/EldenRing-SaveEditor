package schema

type SpellData struct {
	SourceRowID          Fact[uint32] `json:"sourceRowID"`
	IconID               Fact[uint32] `json:"iconID"`
	SortID               Fact[uint32] `json:"sortID"`
	FPCost               Fact[uint32] `json:"fpCost"`
	StaminaCost          Fact[uint32] `json:"staminaCost"`
	MemorySlots          Fact[uint8]  `json:"memorySlots"`
	RequiredIntelligence Fact[uint32] `json:"requiredIntelligence"`
	RequiredFaith        Fact[uint32] `json:"requiredFaith"`
	RequiredArcane       Fact[uint32] `json:"requiredArcane"`
}
