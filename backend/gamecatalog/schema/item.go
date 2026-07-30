package schema

type ItemFamily string

const (
	ItemFamilyWeapon   ItemFamily = "weapon"
	ItemFamilyAshOfWar ItemFamily = "ash_of_war"
)

type ItemDocument struct {
	GameID       Fact[uint32]     `json:"gameID"`
	Family       Fact[ItemFamily] `json:"family"`
	Subcategory  Fact[string]     `json:"subcategory"`
	Presentation ItemPresentation `json:"presentation"`
	Storage      ItemStorage      `json:"storage"`
	Capabilities ItemCapabilities `json:"capabilities"`
	Safety       ItemSafety       `json:"safety"`
	Variants     []ItemVariant    `json:"variants"`
	Weapon       *WeaponData      `json:"weapon"`
	AshOfWar     *AshOfWarData    `json:"ashOfWar"`
}

type ItemPresentation struct {
	CanonicalName Fact[string] `json:"canonicalName"`
	Description   Fact[string] `json:"description"`
	IconPath      Fact[string] `json:"iconPath"`
}

type RecordMode string

const (
	RecordModeQuantityStack     RecordMode = "quantity_stack"
	RecordModeSeparateInstances RecordMode = "separate_instances"
)

type ItemStorage struct {
	RecordMode   Fact[RecordMode] `json:"recordMode"`
	MaxInventory Fact[uint32]     `json:"maxInventory"`
	MaxStorage   Fact[uint32]     `json:"maxStorage"`
}

type ItemSafety struct {
	CutContent Fact[bool] `json:"cutContent"`
	BanRisk    Fact[bool] `json:"banRisk"`
}

type ItemVariant struct {
	GameID      Fact[uint32]   `json:"gameID"`
	Affinity    Fact[Affinity] `json:"affinity"`
	SourceRowID Fact[uint32]   `json:"sourceRowID"`
}
