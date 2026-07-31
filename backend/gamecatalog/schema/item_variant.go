package schema

type ItemVariantKind string

const (
	ItemVariantAffinity        ItemVariantKind = "affinity"
	ItemVariantUpgrade         ItemVariantKind = "upgrade"
	ItemVariantAffinityUpgrade ItemVariantKind = "affinity_upgrade"
)

type ItemVariant struct {
	GameID        Fact[uint32]          `json:"gameID"`
	Kind          Fact[ItemVariantKind] `json:"kind"`
	Affinity      Fact[Affinity]        `json:"affinity"`
	UpgradeLevel  Fact[uint8]           `json:"upgradeLevel"`
	SourceRowID   Fact[uint32]          `json:"sourceRowID"`
	Data          VariantDocumentData   `json:"data"`
	SourceRecords []ParameterRecord     `json:"sourceRecords"`
}

type VariantDocumentData struct {
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
	Unlocks                 []ItemUnlock             `json:"unlocks"`
	RelatedTechnicalRecords []RelatedTechnicalRecord `json:"relatedTechnicalRecords"`
	Weapon                  *WeaponData              `json:"weapon"`
	SpiritAsh               *SpiritAshData           `json:"spiritAsh"`
}

func MaterializeVariant(canonical ItemDocument, variant ItemVariant) ItemDocument {
	data := variant.Data
	canonical.GameID = variant.GameID
	canonical.Family = data.Family
	canonical.Category = data.Category
	canonical.Subcategory = data.Subcategory
	canonical.Flags = data.Flags
	canonical.Presentation = data.Presentation
	canonical.Storage = data.Storage
	canonical.Capabilities = data.Capabilities
	canonical.Safety = data.Safety
	canonical.Acquisition = data.Acquisition
	canonical.Modifiers = data.Modifiers
	canonical.Links = data.Links
	canonical.Unlocks = data.Unlocks
	canonical.RelatedTechnicalRecords = data.RelatedTechnicalRecords
	canonical.SourceRecords = variant.SourceRecords
	canonical.Weapon = data.Weapon
	canonical.SpiritAsh = data.SpiritAsh
	canonical.Armor = nil
	canonical.Talisman = nil
	canonical.AshOfWar = nil
	canonical.Spell = nil
	canonical.Goods = nil
	canonical.Gesture = nil
	return canonical
}
