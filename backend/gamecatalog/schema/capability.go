package schema

type Capability[T any] struct {
	Known      bool       `json:"known"`
	Enabled    bool       `json:"enabled"`
	Rules      *T         `json:"rules"`
	Provenance Provenance `json:"provenance"`
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
	UpgradeModelStandard UpgradeModel = "standard"
)

type UpgradeRules struct {
	Model    UpgradeModel `json:"model"`
	MaxLevel uint8        `json:"maxLevel"`
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
	EquipmentSlotLeftHand  EquipmentSlot = "left_hand"
	EquipmentSlotRightHand EquipmentSlot = "right_hand"
)

type EquipmentRules struct {
	AllowedSlots []EquipmentSlot `json:"allowedSlots"`
}
