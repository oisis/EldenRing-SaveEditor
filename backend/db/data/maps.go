package data

// MapRegionData holds the static definition of a map region.
type MapRegionData struct {
	Name string
	Area string // "Limgrave", "Liurnia", "Altus", "Caelid", "Mountaintops", "Underground", "DLC", "System"
}

// MapSystem contains system-level map display flags.
var MapSystem = map[uint32]MapRegionData{
	62000: {Name: "Allow Map Display", Area: "System"},
	62001: {Name: "Allow Underground Map Display", Area: "System"},
	82001: {Name: "Show Underground", Area: "System"},
	82002: {Name: "Show Shadow Realm Map", Area: "System"},
}

// MapVisible contains map region visibility flags (62xxx).
// Setting these reveals the map texture for each region.
var MapVisible = map[uint32]MapRegionData{
	// Limgrave base tiles
	62004: {Name: "Center", Area: "Limgrave"},
	62005: {Name: "SW", Area: "Limgrave"},
	62006: {Name: "NW", Area: "Limgrave"},
	62007: {Name: "SE", Area: "Limgrave"},
	62008: {Name: "NE", Area: "Limgrave"},
	62009: {Name: "N", Area: "Limgrave"},
	// Limgrave named regions
	62010: {Name: "Limgrave, West", Area: "Limgrave"},
	62011: {Name: "Weeping Peninsula", Area: "Limgrave"},
	62012: {Name: "Limgrave, East", Area: "Limgrave"},
	// Liurnia
	62020: {Name: "Liurnia, East", Area: "Liurnia"},
	62021: {Name: "Liurnia, North", Area: "Liurnia"},
	62022: {Name: "Liurnia, West", Area: "Liurnia"},
	// Altus Plateau
	62030: {Name: "Altus Plateau", Area: "Altus"},
	62031: {Name: "Leyndell, Royal Capital", Area: "Altus"},
	62032: {Name: "Mt. Gelmir", Area: "Altus"},
	// Caelid
	62040: {Name: "Caelid", Area: "Caelid"},
	62041: {Name: "Dragonbarrow", Area: "Caelid"},
	// Mountaintops
	62050: {Name: "Mountaintops of the Giants, West", Area: "Mountaintops"},
	62051: {Name: "Mountaintops of the Giants, East", Area: "Mountaintops"},
	62052: {Name: "Mountaintops of the Giants, North", Area: "Mountaintops"},
	// Underground
	62060: {Name: "Ainsel River", Area: "Underground"},
	62061: {Name: "Lake of Rot", Area: "Underground"},
	62062: {Name: "Mohgwyn Palace", Area: "Underground"},
	62063: {Name: "Siofra River", Area: "Underground"},
	62064: {Name: "Deeproot Depths", Area: "Underground"},
	// DLC — Shadow of the Erdtree
	62080: {Name: "Gravesite Plain", Area: "DLC"},
	62081: {Name: "Scadu Altus", Area: "DLC"},
	62082: {Name: "Southern Shore", Area: "DLC"},
	62083: {Name: "Rauh Ruins", Area: "DLC"},
	62084: {Name: "Abyss", Area: "DLC"},
	// Catacombs / Caves
	62102: {Name: "Fringefolk Hero's Cave", Area: "Limgrave"},
	62103: {Name: "Stormfoot Catacombs", Area: "Limgrave"},
}

// MapAcquired contains map fragment acquisition flags (63xxx).
// Setting these records that the physical Map Fragment item was picked up.
var MapAcquired = map[uint32]MapRegionData{
	63010: {Name: "Limgrave, West", Area: "Limgrave"},
	63011: {Name: "Weeping Peninsula", Area: "Limgrave"},
	63012: {Name: "Limgrave, East", Area: "Limgrave"},
	63020: {Name: "Liurnia, East", Area: "Liurnia"},
	63021: {Name: "Liurnia, North", Area: "Liurnia"},
	63022: {Name: "Liurnia, West", Area: "Liurnia"},
	63030: {Name: "Altus Plateau", Area: "Altus"},
	63031: {Name: "Leyndell, Royal Capital", Area: "Altus"},
	63032: {Name: "Mt. Gelmir", Area: "Altus"},
	63040: {Name: "Caelid", Area: "Caelid"},
	63041: {Name: "Dragonbarrow", Area: "Caelid"},
	63050: {Name: "Mountaintops of the Giants, West", Area: "Mountaintops"},
	63051: {Name: "Mountaintops of the Giants, East", Area: "Mountaintops"},
	63052: {Name: "Consecrated Snowfield", Area: "Mountaintops"},
	63060: {Name: "Ainsel River", Area: "Underground"},
	63061: {Name: "Lake of Rot", Area: "Underground"},
	63062: {Name: "Mohgwyn Palace", Area: "Underground"},
	63063: {Name: "Siofra River", Area: "Underground"},
	63064: {Name: "Deeproot Depths", Area: "Underground"},
	63080: {Name: "Gravesite Plain", Area: "DLC"},
	63081: {Name: "Scadu Altus", Area: "DLC"},
	63082: {Name: "Southern Shore", Area: "DLC"},
	63083: {Name: "Rauh Ruins", Area: "DLC"},
	63084: {Name: "Abyss", Area: "DLC"},
}

// MapPOIRange defines a range of event flag IDs for point-of-interest discovery.
type MapPOIRange struct {
	Start uint32
	End   uint32
	Area  string
}

// MapPOIRanges defines the event flag ranges for map point discovery.
// Setting all flags in these ranges reveals POI icons on the map.
var MapPOIRanges = []MapPOIRange{
	{Start: 62100, End: 62199, Area: "Limgrave"},
	{Start: 62200, End: 62299, Area: "Liurnia"},
	{Start: 62300, End: 62399, Area: "Altus"},
	{Start: 62400, End: 62499, Area: "Caelid"},
	{Start: 62500, End: 62599, Area: "Mountaintops"},
	{Start: 62600, End: 62699, Area: "Underground"},
	{Start: 62700, End: 62799, Area: "Other"},
}
