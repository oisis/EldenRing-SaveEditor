package schema

type SpellData struct {
	SourceRowID             Fact[uint32]  `json:"sourceRowID"`
	IconID                  Fact[uint32]  `json:"iconID"`
	SortID                  Fact[uint32]  `json:"sortID"`
	SortIDSFV               *Fact[uint32] `json:"sortID-sfv,omitempty"`
	FPCost                  Fact[uint32]  `json:"fpCost"`
	FPCostSFV               *Fact[uint32] `json:"fpCost-sfv,omitempty"`
	StaminaCost             Fact[uint32]  `json:"staminaCost"`
	MemorySlots             Fact[uint8]   `json:"memorySlots"`
	MemorySlotsSFV          *Fact[uint8]  `json:"memorySlots-sfv,omitempty"`
	RequiredIntelligence    Fact[uint32]  `json:"requiredIntelligence"`
	RequiredIntelligenceSFV *Fact[uint32] `json:"requiredIntelligence-sfv,omitempty"`
	RequiredFaith           Fact[uint32]  `json:"requiredFaith"`
	RequiredFaithSFV        *Fact[uint32] `json:"requiredFaith-sfv,omitempty"`
	RequiredArcane          Fact[uint32]  `json:"requiredArcane"`
	RequiredArcaneSFV       *Fact[uint32] `json:"requiredArcane-sfv,omitempty"`
}
