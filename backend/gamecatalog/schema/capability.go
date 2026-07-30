package schema

type Capability[T any] struct {
	Known      bool
	Enabled    bool
	Rules      *T
	Provenance Provenance
}

type ItemCapabilities struct {
	Upgrade       Capability[UpgradeRules]
	Infusion      Capability[InfusionRules]
	AshOfWarMount Capability[AshOfWarMountRules]
	Stack         Capability[StackRules]
	Equipment     Capability[EquipmentRules]
}

type UpgradeModel string

const (
	UpgradeModelStandard UpgradeModel = "standard"
)

type UpgradeRules struct {
	Model    UpgradeModel
	MaxLevel uint8
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
	AllowedAffinities []Affinity
}

type AshOfWarMountMode string

const (
	AshOfWarMountModeCustom AshOfWarMountMode = "custom"
)

type AshOfWarMountRules struct {
	Mode             AshOfWarMountMode
	WeaponType       string
	CompatibilityBit uint8
}

type StackRules struct {
	MaxPerStack uint32
}

type EquipmentSlot string

const (
	EquipmentSlotLeftHand  EquipmentSlot = "left_hand"
	EquipmentSlotRightHand EquipmentSlot = "right_hand"
)

type EquipmentRules struct {
	AllowedSlots []EquipmentSlot
}
