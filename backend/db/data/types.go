package data

// ItemData represents the metadata for an item in the game database.
type ItemData struct {
	Name         string
	Category     string
	MaxInventory uint32
	MaxStorage   uint32
	MaxUpgrade   uint32
	IconPath     string
	Flags        []string
}

// WeaponStats holds base stats for a weapon (before upgrades/infusions).
type WeaponStats struct {
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

// ArmorStats holds damage negation and resistance values for armor.
type ArmorStats struct {
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

// SpellStats holds FP cost, slot count, and attribute requirements for spells.
type SpellStats struct {
	FPCost uint32
	Slots  uint32
	ReqInt uint32
	ReqFai uint32
	ReqArc uint32
}

// ItemDescription holds an item's in-game description text and optional stats.
type ItemDescription struct {
	Description string
	Weight      float64
	Weapon      *WeaponStats
	Armor       *ArmorStats
	Spell       *SpellStats
}

// Descriptions maps item IDs to their descriptions and stats.
// Populated by generated code in descriptions.go.
var Descriptions map[uint32]ItemDescription
