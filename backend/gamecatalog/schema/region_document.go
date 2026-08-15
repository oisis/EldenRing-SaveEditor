package schema

// RegionDocument declares one curated region of the invasion / blue-summon
// allowlist. RegionID is the PlayRegionParam row ID the per-slot UnlockedRegions
// list stores, Name is the region name and Area is the grouping the region is
// presented under. Each fact carries its own provenance; none is derived from
// another.
//
// RegionID stays an internal save-format detail: it is what the backend matches
// a stored ID against, and it is never part of the GetRegions result.
type RegionDocument struct {
	RegionID Fact[uint32] `json:"regionID"`
	Name     Fact[string] `json:"name"`
	Area     Fact[string] `json:"area"`
}
