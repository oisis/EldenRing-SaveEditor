package schema

type ItemFamily string

const (
	ItemFamilyWeapon    ItemFamily = "weapon"
	ItemFamilyArmor     ItemFamily = "armor"
	ItemFamilyTalisman  ItemFamily = "talisman"
	ItemFamilyAshOfWar  ItemFamily = "ash_of_war"
	ItemFamilySpell     ItemFamily = "spell"
	ItemFamilySpiritAsh ItemFamily = "spirit_ash"
	ItemFamilyGoods     ItemFamily = "goods"
	ItemFamilyGesture   ItemFamily = "gesture"
)

type ItemDocument struct {
	GameID                  Fact[uint32]             `json:"gameID"`
	Family                  Fact[ItemFamily]         `json:"family"`
	Category                Fact[string]             `json:"category"`
	Subcategory             Fact[string]             `json:"subcategory"`
	Flags                   Fact[[]string]           `json:"flags"`
	Presentation            ItemPresentation         `json:"presentation"`
	Storage                 ItemStorage              `json:"storage"`
	Capabilities            ItemCapabilities         `json:"capabilities"`
	Safety                  ItemSafety               `json:"safety"`
	Acquisition             ItemAcquisition          `json:"acquisition"`
	Modifiers               ItemModifiers            `json:"modifiers"`
	Links                   ItemLinks                `json:"links"`
	Variants                []ItemVariant            `json:"variants"`
	Aliases                 []ItemAlias              `json:"aliases"`
	Unlocks                 []ItemUnlock             `json:"unlocks"`
	RelatedTechnicalRecords []RelatedTechnicalRecord `json:"relatedTechnicalRecords"`
	SourceRecords           []ParameterRecord        `json:"sourceRecords"`
	Weapon                  *WeaponData              `json:"weapon"`
	Armor                   *ArmorData               `json:"armor"`
	Talisman                *TalismanData            `json:"talisman"`
	AshOfWar                *AshOfWarData            `json:"ashOfWar"`
	Spell                   *SpellData               `json:"spell"`
	SpiritAsh               *SpiritAshData           `json:"spiritAsh"`
	Goods                   *GoodsData               `json:"goods"`
	Gesture                 *GestureData             `json:"gesture"`
}

type ItemPresentation struct {
	DisplayName   Fact[string]     `json:"displayName"`
	CanonicalName Fact[string]     `json:"canonicalName"`
	Caption       Fact[string]     `json:"caption"`
	Description   Fact[string]     `json:"description"`
	Location      Fact[string]     `json:"location"`
	IconPath      Fact[string]     `json:"iconPath"`
	TextMetadata  ItemTextMetadata `json:"textMetadata"`
}

type ItemTextMetadata struct {
	DisplayNameSource Fact[string] `json:"displayNameSource"`
	CanonicalSource   Fact[string] `json:"canonicalSource"`
	CaptionSource     Fact[string] `json:"captionSource"`
	DescriptionSource Fact[string] `json:"descriptionSource"`
	LocationSource    Fact[string] `json:"locationSource"`
	DLCSource         Fact[string] `json:"dlcSource"`
	Notes             Fact[string] `json:"notes"`
}

type RecordMode string

const (
	RecordModeQuantityStack     RecordMode = "quantity_stack"
	RecordModeSeparateInstances RecordMode = "separate_instances"
)

type ItemStorage struct {
	RecordMode          Fact[RecordMode] `json:"recordMode"`
	MaxInventory        Fact[uint32]     `json:"maxInventory"`
	MaxInventorySFV     *Fact[uint32]    `json:"maxInventory-sfv,omitempty"`
	MaxStorage          Fact[uint32]     `json:"maxStorage"`
	MaxStorageSFV       *Fact[uint32]    `json:"maxStorage-sfv,omitempty"`
	GameMaxInventory    Fact[uint32]     `json:"gameMaxInventory"`
	GameMaxInventorySFV *Fact[uint32]    `json:"gameMaxInventory-sfv,omitempty"`
	GameMaxStorage      Fact[uint32]     `json:"gameMaxStorage"`
	GameMaxStorageSFV   *Fact[uint32]    `json:"gameMaxStorage-sfv,omitempty"`
}

type ItemSafety struct {
	CutContent   Fact[bool] `json:"cutContent"`
	BanRisk      Fact[bool] `json:"banRisk"`
	DLC          Fact[bool] `json:"dlc"`
	NoDatabase   Fact[bool] `json:"noDatabase"`
	ScalesWithNG Fact[bool] `json:"scalesWithNG"`
}

type ItemAlias struct {
	GameID        Fact[uint32]      `json:"gameID"`
	SourceRecords []ParameterRecord `json:"sourceRecords"`
}

type ItemUnlock struct {
	Kind        Fact[string] `json:"kind"`
	EventFlagID Fact[uint32] `json:"eventFlagID"`
	Name        Fact[string] `json:"name"`
	Category    Fact[string] `json:"category"`
}
