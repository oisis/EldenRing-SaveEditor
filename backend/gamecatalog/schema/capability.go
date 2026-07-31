package schema

type Capability[T any] struct {
	Known         bool         `json:"known"`
	Enabled       bool         `json:"enabled"`
	Rules         *T           `json:"rules"`
	Provenance    Provenance   `json:"provenance"`
	RulesEvidence []Provenance `json:"rulesEvidence,omitempty"`
}

type ItemCapabilities struct {
	Upgrade       Capability[UpgradeRules]       `json:"upgrade"`
	Infusion      Capability[InfusionRules]      `json:"infusion"`
	AshOfWarMount Capability[AshOfWarMountRules] `json:"ashOfWarMount"`
	Stack         Capability[StackRules]         `json:"stack"`
	Equipment     Capability[EquipmentRules]     `json:"equipment"`
}

type UpgradeModel string

const (
	UpgradeModelStandard       UpgradeModel = "standard"
	UpgradeModelSomber         UpgradeModel = "somber"
	UpgradeModelGraveGlovewort UpgradeModel = "grave_glovewort"
	UpgradeModelGhostGlovewort UpgradeModel = "ghost_glovewort"
)

type UpgradeRules struct {
	Model       UpgradeModel `json:"model"`
	MaxLevel    uint8        `json:"maxLevel"`
	MaxLevelSFV *Fact[uint8] `json:"maxLevel-sfv,omitempty"`
}

type Affinity string

const (
	AffinityStandard  Affinity = "standard"
	AffinityHeavy     Affinity = "heavy"
	AffinityKeen      Affinity = "keen"
	AffinityQuality   Affinity = "quality"
	AffinityFire      Affinity = "fire"
	AffinityFlameArt  Affinity = "flame_art"
	AffinityLightning Affinity = "lightning"
	AffinitySacred    Affinity = "sacred"
	AffinityMagic     Affinity = "magic"
	AffinityCold      Affinity = "cold"
	AffinityPoison    Affinity = "poison"
	AffinityBlood     Affinity = "blood"
	AffinityOccult    Affinity = "occult"
)

type InfusionRules struct {
	AllowedAffinities []Affinity `json:"allowedAffinities"`
}

type AshOfWarMountMode string

const (
	AshOfWarMountModeCustom AshOfWarMountMode = "custom"
)

type AshOfWarMountRules struct {
	Mode             AshOfWarMountMode `json:"mode"`
	WeaponType       string            `json:"weaponType"`
	CompatibilityBit uint8             `json:"compatibilityBit"`
}

type StackRules struct {
	MaxPerStack uint32 `json:"maxPerStack"`
}

type EquipmentSlot string

const (
	EquipmentSlotLeftHand    EquipmentSlot = "left_hand"
	EquipmentSlotRightHand   EquipmentSlot = "right_hand"
	EquipmentSlotArrow       EquipmentSlot = "arrow"
	EquipmentSlotBolt        EquipmentSlot = "bolt"
	EquipmentSlotHead        EquipmentSlot = "head"
	EquipmentSlotChest       EquipmentSlot = "chest"
	EquipmentSlotArms        EquipmentSlot = "arms"
	EquipmentSlotLegs        EquipmentSlot = "legs"
	EquipmentSlotTalisman    EquipmentSlot = "talisman"
	EquipmentSlotSpellMemory EquipmentSlot = "spell_memory"
	EquipmentSlotQuickItem   EquipmentSlot = "quick_item"
	EquipmentSlotPouch       EquipmentSlot = "pouch"
	EquipmentSlotPhysick     EquipmentSlot = "physick"
)

type EquipmentRules struct {
	AllowedSlots []EquipmentSlot `json:"allowedSlots"`
}
