package schema

// MapRegionDocument declares one entry of the curated map visibility table. Name
// is the map region name, AreaLabel is the curated label the region is presented
// and grouped under and VisibleEventFlagID is the event flag whose set bit makes
// the map texture of that region visible in a save. Each fact carries its own
// provenance; none is derived from another.
//
// AreaLabel is a presentation and grouping label only. It is deliberately not a
// ResourceRef and never resolves to a RegionDocument: the curated table declares
// a plain text area and nothing else, so joining the two vocabularies would
// invent a relation no evidence supports.
//
// The table covers the confirmed safe visibility flags alone. The legacy
// sub-region flags that produce black map tiles when set outside the game's own
// discovery flow, the system-level map display flags, and the transient map
// fragment pickup flags are all deliberately absent because they do not belong
// to the safe region-visibility table exposed by this document.
type MapRegionDocument struct {
	Name               Fact[string] `json:"name"`
	AreaLabel          Fact[string] `json:"areaLabel"`
	VisibleEventFlagID Fact[uint32] `json:"visibleEventFlagID"`
}

// mapRegionFlagsPerBlock is the block size the event flag identifiers are split
// by, and mapRegionFlagBlock is the single block the curated map visibility
// table uses. Its BST position is confirmed in SaveForge 1.5.8 and 1.6.8 alike.
const (
	mapRegionFlagsPerBlock uint32 = 1000
	mapRegionFlagBlock     uint32 = 62
)

// IsConfirmedMapRegionFlag reports whether one visibility event flag lies in the
// block the curated table confirms. It is the single source of that rule for the
// catalog validation and for the generator that produces the documents.
func IsConfirmedMapRegionFlag(eventFlagID uint32) bool {
	return eventFlagID/mapRegionFlagsPerBlock == mapRegionFlagBlock
}
