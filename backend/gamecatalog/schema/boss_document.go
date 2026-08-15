package schema

// BossDocument declares one boss encounter of the curated Bosses table. Name is
// the boss name, RegionLabel is the curated label the encounter is presented and
// grouped under, EncounterType keeps the confirmed `main` or `field`
// classification of the curated table, Remembrance records whether the boss
// drops a Remembrance item and DefeatEventFlagID is the synchronized event flag
// whose set bit records the defeat in a save. Each fact carries its own
// provenance; none is derived from another.
//
// RegionLabel is a presentation and grouping label only. It is deliberately not
// a ResourceRef and never resolves to a RegionDocument: the curated table
// declares a plain text region and nothing else, so joining the two vocabularies
// would invent a relation no evidence supports.
//
// The table covers only the bosses with a confirmed synchronized defeat flag.
// The roughly 97 open-world field bosses whose defeat is recorded solely by a
// per-map flag are deliberately absent, because their state cannot be resolved
// from this bitfield at all.
type BossDocument struct {
	Name              Fact[string] `json:"name"`
	RegionLabel       Fact[string] `json:"regionLabel"`
	EncounterType     Fact[string] `json:"encounterType"`
	Remembrance       Fact[bool]   `json:"remembrance"`
	DefeatEventFlagID Fact[uint32] `json:"defeatEventFlagID"`
}

// The confirmed encounter types of the curated table.
const (
	BossEncounterTypeMain  = "main"
	BossEncounterTypeField = "field"
)

// bossFlagsPerBlock is the block size the event flag identifiers are split by,
// and bossFlagBlock is the single block the curated Bosses table uses. Its BST
// position is confirmed in SaveForge 1.5.8 and 1.6.8 alike.
const (
	bossFlagsPerBlock uint32 = 1000
	bossFlagBlock     uint32 = 9
)

// IsConfirmedBossFlag reports whether one defeat event flag lies in the block
// the curated Bosses table confirms. It is the single source of that rule for
// the catalog validation and for the generator that produces the documents.
func IsConfirmedBossFlag(eventFlagID uint32) bool {
	return eventFlagID/bossFlagsPerBlock == bossFlagBlock
}
