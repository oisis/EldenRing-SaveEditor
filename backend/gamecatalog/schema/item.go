package schema

type ItemFamily string

const (
	ItemFamilyWeapon   ItemFamily = "weapon"
	ItemFamilyAshOfWar ItemFamily = "ash_of_war"
)

type ItemDocument struct {
	GameID       Fact[uint32]
	Family       Fact[ItemFamily]
	Subcategory  Fact[string]
	Presentation ItemPresentation
	Storage      ItemStorage
	Capabilities ItemCapabilities
	Safety       ItemSafety
	Variants     []ItemVariant
	Weapon       *WeaponData
	AshOfWar     *AshOfWarData
}

type ItemPresentation struct {
	CanonicalName Fact[string]
	Description   Fact[string]
	IconPath      Fact[string]
}

type RecordMode string

const (
	RecordModeQuantityStack     RecordMode = "quantity_stack"
	RecordModeSeparateInstances RecordMode = "separate_instances"
)

type ItemStorage struct {
	RecordMode   Fact[RecordMode]
	MaxInventory Fact[uint32]
	MaxStorage   Fact[uint32]
}

type ItemSafety struct {
	CutContent Fact[bool]
	BanRisk    Fact[bool]
}

type ItemVariant struct {
	GameID      Fact[uint32]
	Affinity    Fact[Affinity]
	SourceRowID Fact[uint32]
}
