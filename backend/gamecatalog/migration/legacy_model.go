package migration

type seed struct {
	ID                    uint32
	HasLegacyItem         bool
	RegulationOnlyVariant bool
	Category              string
	Name                  string
	Subcategory           string
	MaxInventory          uint32
	MaxStorage            uint32
	GameMaxInventory      uint32
	GameMaxStorage        uint32
	GameMaxInventoryKnown bool
	GameMaxStorageKnown   bool
	MaxUpgrade            uint32
	IconPath              string
	Flags                 []string
	Text                  *textSeed
	Description           *descriptionSeed
	GameLimits            *gameLimitsSeed
	WeaponStats           *weaponStatsSeed
	Weight                *float64
	WeaponEdit            *weaponEditSeed
	AoWCompatMask         *uint64
	SpellMemory           *uint8
	SortKey               *sortKeySeed
	Acquisition           acquisitionSeed
	EquipLoad             *equipLoadSeed
	Unlocks               []unlockSeed
	Links                 linksSeed
	AoWCompatibleClasses  []string
}

type legacySnapshot struct {
	Items            []seed
	Aliases          []aliasSeed
	GestureSlots     []gestureSlotSeed
	SwordArtsNames   []swordArtsNameSeed
	TechnicalRecords []technicalRecordSeed
}

type textSeed struct {
	DisplayName       string
	CanonicalName     string
	Caption           string
	Description       string
	Location          string
	DisplayNameSource string
	CanonicalSource   string
	CaptionSource     string
	DescriptionSource string
	LocationSource    string
	DLCSource         string
	Notes             string
}

type gameLimitsSeed struct {
	MaxInventory   uint32
	MaxStorage     uint32
	InventoryKnown bool
	StorageKnown   bool
}

type descriptionSeed struct {
	Description string
	Location    string
	Weight      float64
	Weapon      *legacyWeaponStats
	Armor       *legacyArmorStats
	Spell       *legacySpellStats
}

type legacyWeaponStats struct {
	Weight     float64
	PhysDamage uint32
	MagDamage  uint32
	FireDamage uint32
	LitDamage  uint32
	HolyDamage uint32
	ScaleStr   uint32
	ScaleDex   uint32
	ScaleInt   uint32
	ScaleFai   uint32
	ReqStr     uint32
	ReqDex     uint32
	ReqInt     uint32
	ReqFai     uint32
	ReqArc     uint32
}

type legacyArmorStats struct {
	Weight     float64
	Physical   float64
	Strike     float64
	Slash      float64
	Pierce     float64
	Magic      float64
	Fire       float64
	Lightning  float64
	Holy       float64
	Immunity   uint32
	Robustness uint32
	Focus      uint32
	Vitality   uint32
	Poise      float64
}

type legacySpellStats struct {
	FPCost uint32
	Slots  uint32
	ReqInt uint32
	ReqFai uint32
	ReqArc uint32
}

type weaponEditSeed struct {
	WepType           uint16
	GemMountType      uint8
	CanChangeAffinity bool
	CompatibilityBit  *uint8
}

type unlockSeed struct {
	Kind     string
	FlagID   uint32
	Name     string
	Category string
}

type sortKeySeed struct {
	SortID      uint32
	SortGroupID uint8
}

type acquisitionSeed struct {
	RequiredContainerID   *uint32
	IsContainer           bool
	ContainerPickupFlags  []uint32
	ContainerVendorFlags  []uint32
	BolsteringPickupFlags []uint32
	WorldPickupFlagID     *uint32
	CompanionEventFlagIDs []uint32
}

type equipLoadSeed struct {
	EnduranceBonus int32
	EquipLoadRate  float64
}

type aliasSeed struct {
	AliasID     uint32
	CanonicalID uint32
}

type gestureSlotSeed struct {
	SlotID   uint32
	ItemID   uint32
	Name     string
	Category string
	Flags    []string
}

type linksSeed struct {
	AboutTutorialID   *uint32
	RelatedEventFlags []relatedEventFlagSeed
	RelatedItems      []relatedItemSeed
	WhetbladeName     string
	MapFragment       *mapFragmentSeed
}

type relatedEventFlagSeed struct {
	Kind   string
	FlagID uint32
}

type relatedItemSeed struct {
	Kind   string
	ItemID uint32
}

type mapFragmentSeed struct {
	Name           string
	Area           string
	AcquiredFlagID uint32
}

type technicalRecordSeed struct {
	ID          uint32
	Description descriptionSeed
	GameLimits  gameLimitsSeed
}

type swordArtsNameSeed struct {
	ID   int32
	Name string
}
