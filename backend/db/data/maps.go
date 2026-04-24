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

// MapVisible contains safe map region visibility flags (62xxx).
// Setting these reveals the map texture for each region.
// Only includes flags verified as safe — see MapUnsafe for risky sub-region flags.
var MapVisible = map[uint32]MapRegionData{
	// Limgrave
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
	62052: {Name: "Consecrated Snowfield", Area: "Mountaintops"},
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
	// Dungeon maps
	62102: {Name: "Fringefolk Hero's Cave", Area: "Limgrave"},
	62103: {Name: "Stormfoot Catacombs", Area: "Limgrave"},
}

// MapUnsafe contains sub-region visibility flags that can cause black map tiles
// when set without the game's normal discovery flow. Shown in UI but excluded
// from "Reveal All" to prevent visual corruption.
var MapUnsafe = map[uint32]MapRegionData{
	62004: {Name: "Center (sub-region)", Area: "Limgrave"},
	62005: {Name: "SW (sub-region)", Area: "Limgrave"},
	62006: {Name: "NW (sub-region)", Area: "Limgrave"},
	62007: {Name: "SE (sub-region)", Area: "Limgrave"},
	62008: {Name: "NE (sub-region)", Area: "Limgrave"},
	62009: {Name: "N (sub-region)", Area: "Limgrave"},
	62053: {Name: "Mountaintops, North (sub-region)", Area: "Mountaintops"},
	62065: {Name: "Underground (sub-region)", Area: "Underground"},
}

// MapFragmentItems maps visible flag IDs (62xxx) to their corresponding
// map fragment inventory item IDs (0x400021xx / 0x401EAxxx).
// Used by SetMapRegion/RevealAllMap to add map items to inventory.
var MapFragmentItems = map[uint32]uint32{
	// Base game
	62010: 0x40002198, // Limgrave, West
	62011: 0x40002199, // Weeping Peninsula
	62012: 0x4000219A, // Limgrave, East
	62020: 0x4000219B, // Liurnia, East
	62021: 0x4000219C, // Liurnia, North
	62022: 0x4000219D, // Liurnia, West
	62030: 0x4000219E, // Altus Plateau
	62031: 0x4000219F, // Leyndell, Royal Capital
	62032: 0x400021A0, // Mt. Gelmir
	62040: 0x400021A1, // Caelid
	62041: 0x400021A2, // Dragonbarrow
	62050: 0x400021A3, // Mountaintops of the Giants, West
	62051: 0x400021A4, // Mountaintops of the Giants, East
	62052: 0x400021AA, // Consecrated Snowfield
	62060: 0x400021A5, // Ainsel River
	62061: 0x400021A6, // Lake of Rot
	62062: 0x400021A8, // Mohgwyn Palace
	62063: 0x400021A7, // Siofra River
	62064: 0x400021A9, // Deeproot Depths
	// DLC — Shadow of the Erdtree
	62080: 0x401EA618, // Gravesite Plain
	62081: 0x401EA619, // Scadu Altus
	62082: 0x401EA61A, // Southern Shore
	62083: 0x401EA61B, // Rauh Ruins
	62084: 0x401EA61C, // Abyss
}

// MapAcquired contains map fragment acquisition flags (63xxx).
// These are transient "pickup notification pending" triggers — the game clears them
// after showing the "Map Fragment acquired" popup. NOT used for map visibility or items.
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

// IsDLCMapFlag returns true if the visible flag ID belongs to a DLC (Shadow of the Erdtree) map region.
func IsDLCMapFlag(flagID uint32) bool {
	return (flagID >= 62080 && flagID <= 62084) ||
		(flagID >= 62800 && flagID <= 62999)
}
