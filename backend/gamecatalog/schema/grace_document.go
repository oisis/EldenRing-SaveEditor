package schema

// GraceDocument declares one Site of Grace of the curated Graces table. Name is
// the grace name, RegionLabel is the curated label the grace is presented and
// grouped under, VisitEventFlagID is the event flag whose set bit records the
// visit in a save, BossArena records whether the grace belongs to a boss arena,
// DungeonType names the sealed-entrance dungeon family the grace belongs to and
// DoorEventFlagID is the overworld ObjAct flag that opens that entrance. Each
// fact carries its own provenance; none is derived from another.
//
// RegionLabel is a presentation and grouping label only. It is deliberately not
// a ResourceRef and never resolves to a RegionDocument: the two vocabularies are
// separate curated lists and joining them would invent a relation no evidence
// supports.
//
// DoorEventFlagID is private catalog data. It is part of the same confirmed
// record and is stored so the document stays complete, but no getter exposes it.
type GraceDocument struct {
	Name             Fact[string] `json:"name"`
	RegionLabel      Fact[string] `json:"regionLabel"`
	VisitEventFlagID Fact[uint32] `json:"visitEventFlagID"`
	BossArena        Fact[bool]   `json:"bossArena"`
	DungeonType      Fact[string] `json:"dungeonType"`
	DoorEventFlagID  Fact[uint32] `json:"doorEventFlagID"`
}

// The confirmed dungeon families of the curated table. An empty value is the
// confirmed value of a grace that is not behind a sealed dungeon entrance, so it
// is a known fact and not a missing one.
const (
	GraceDungeonTypeNone      = ""
	GraceDungeonTypeCatacomb  = "catacomb"
	GraceDungeonTypeHeroGrave = "hero_grave"
)

// graceFlagsPerBlock is the block size the event flag identifiers are split by,
// and graceFlagBlocks are the blocks the curated Graces table actually uses.
// Block 75 exists in the bitfield layout but carries no grace of the table, so
// it is deliberately absent: a flag may only be served from a block the curated
// data itself proves.
const graceFlagsPerBlock uint32 = 1000

var graceFlagBlocks = [...]uint32{71, 72, 73, 74, 76}

// IsConfirmedGraceFlag reports whether one visit event flag lies in a block the
// curated Graces table confirms. It is the single source of that rule for the
// catalog validation and for the generator that produces the documents.
func IsConfirmedGraceFlag(eventFlagID uint32) bool {
	block := eventFlagID / graceFlagsPerBlock
	for _, confirmed := range graceFlagBlocks {
		if block == confirmed {
			return true
		}
	}
	return false
}
