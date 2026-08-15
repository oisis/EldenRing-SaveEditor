package schema

// SummoningPoolDocument declares one summoning pool (Martyr Effigy). Name is the
// pool name, RegionLabel is the curated label the pool is presented and grouped
// under, and ActivationEventFlagID is the event flag whose set bit records the
// activation in a save. Each fact carries its own provenance; none is derived
// from another.
//
// RegionLabel is a presentation and grouping label only. It is deliberately not
// a ResourceRef and never resolves to a RegionDocument: the two vocabularies are
// separate curated lists and joining them would invent a relation no evidence
// supports.
type SummoningPoolDocument struct {
	Name                  Fact[string] `json:"name"`
	RegionLabel           Fact[string] `json:"regionLabel"`
	ActivationEventFlagID Fact[uint32] `json:"activationEventFlagID"`
}

// Confirmed bounds of the one event-flag block summoning pools live in. Every
// activation flag of the curated list lies inside block 670, so a flag outside
// it is rejected instead of being served from a block no evidence places.
const (
	SummoningPoolFlagBlockFirst uint32 = 670000
	SummoningPoolFlagBlockLast  uint32 = 670999
)
