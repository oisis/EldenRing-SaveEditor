package schema

type WeaponData struct {
	SourceRowID       Fact[uint32]
	WeaponTypeID      Fact[uint16]
	Weight            Fact[float32]
	AttackPhysical    Fact[uint32]
	RequiredStrength  Fact[uint16]
	RequiredDexterity Fact[uint16]
	Critical          Fact[uint16]
}
