package schema

type ItemAcquisition struct {
	RequiredContainerID     Fact[uint32]   `json:"requiredContainerID"`
	IsContainer             Fact[bool]     `json:"isContainer"`
	ContainerPickupFlagIDs  Fact[[]uint32] `json:"containerPickupFlagIDs"`
	ContainerVendorFlagIDs  Fact[[]uint32] `json:"containerVendorFlagIDs"`
	BolsteringPickupFlagIDs Fact[[]uint32] `json:"bolsteringPickupFlagIDs"`
	WorldPickupFlagID       Fact[uint32]   `json:"worldPickupFlagID"`
	CompanionEventFlagIDs   Fact[[]uint32] `json:"companionEventFlagIDs"`
}

type ItemModifiers struct {
	EquipLoad *EquipLoadModifier `json:"equipLoad,omitempty"`
}

type EquipLoadModifier struct {
	EnduranceBonus Fact[int32]   `json:"enduranceBonus"`
	EquipLoadRate  Fact[float64] `json:"equipLoadRate"`
}

type ItemDescriptionRecord struct {
	Description Fact[string]        `json:"description"`
	Location    Fact[string]        `json:"location"`
	Weight      Fact[float64]       `json:"weight"`
	Weapon      *CuratedWeaponStats `json:"weapon"`
	Armor       *CuratedArmorStats  `json:"armor"`
	Spell       *CuratedSpellStats  `json:"spell"`
}

type CuratedWeaponStats struct {
	Weight               Fact[float64] `json:"weight"`
	AttackPhysical       Fact[uint32]  `json:"attackPhysical"`
	AttackMagic          Fact[uint32]  `json:"attackMagic"`
	AttackFire           Fact[uint32]  `json:"attackFire"`
	AttackLightning      Fact[uint32]  `json:"attackLightning"`
	AttackHoly           Fact[uint32]  `json:"attackHoly"`
	ScalingStrength      Fact[uint32]  `json:"scalingStrength"`
	ScalingDexterity     Fact[uint32]  `json:"scalingDexterity"`
	ScalingIntelligence  Fact[uint32]  `json:"scalingIntelligence"`
	ScalingFaith         Fact[uint32]  `json:"scalingFaith"`
	RequiredStrength     Fact[uint32]  `json:"requiredStrength"`
	RequiredDexterity    Fact[uint32]  `json:"requiredDexterity"`
	RequiredIntelligence Fact[uint32]  `json:"requiredIntelligence"`
	RequiredFaith        Fact[uint32]  `json:"requiredFaith"`
	RequiredArcane       Fact[uint32]  `json:"requiredArcane"`
}

type CuratedArmorStats struct {
	Weight     Fact[float64] `json:"weight"`
	Physical   Fact[float64] `json:"physical"`
	Strike     Fact[float64] `json:"strike"`
	Slash      Fact[float64] `json:"slash"`
	Pierce     Fact[float64] `json:"pierce"`
	Magic      Fact[float64] `json:"magic"`
	Fire       Fact[float64] `json:"fire"`
	Lightning  Fact[float64] `json:"lightning"`
	Holy       Fact[float64] `json:"holy"`
	Immunity   Fact[uint32]  `json:"immunity"`
	Robustness Fact[uint32]  `json:"robustness"`
	Focus      Fact[uint32]  `json:"focus"`
	Vitality   Fact[uint32]  `json:"vitality"`
	Poise      Fact[float64] `json:"poise"`
}

type CuratedSpellStats struct {
	FPCost               Fact[uint32] `json:"fpCost"`
	MemorySlots          Fact[uint32] `json:"memorySlots"`
	RequiredIntelligence Fact[uint32] `json:"requiredIntelligence"`
	RequiredFaith        Fact[uint32] `json:"requiredFaith"`
	RequiredArcane       Fact[uint32] `json:"requiredArcane"`
}
