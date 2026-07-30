package schema

type WeaponData struct {
	SourceRowID       Fact[uint32]  `json:"sourceRowID"`
	WeaponTypeID      Fact[uint16]  `json:"weaponTypeID"`
	Weight            Fact[float32] `json:"weight"`
	AttackPhysical    Fact[uint32]  `json:"attackPhysical"`
	RequiredStrength  Fact[uint16]  `json:"requiredStrength"`
	RequiredDexterity Fact[uint16]  `json:"requiredDexterity"`
	Critical          Fact[uint16]  `json:"critical"`
}
