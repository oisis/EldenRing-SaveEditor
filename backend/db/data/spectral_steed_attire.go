package data

// SpectralSteedAttire is one selectable Torrent appearance.
//
// Confirmed on PC native 1.17 saves: event flags 6700-6703 are mutually
// exclusive — exactly one of them is set once the player has ever opened the
// Spectral Steed Attire menu. Saves created before 1.17 have all four cleared;
// that legacy state must not be interpreted as "Default is active".
//
// ItemID is 0 for the default appearance, which needs no inventory item. The
// three attires reference existing Regulation 1.17 key items; their icons stay
// in KeyItems so this table never duplicates icon data.
type SpectralSteedAttire struct {
	FlagID uint32
	Name   string
	ItemID uint32
}

// SpectralSteedAttires lists the four appearances in menu order.
var SpectralSteedAttires = []SpectralSteedAttire{
	{FlagID: SpectralSteedAttireDefaultFlag, Name: "Default Appearance"},
	{FlagID: 6701, Name: "Tree Sentinel Spectral Steed Attire", ItemID: 0x401EAA00},
	{FlagID: 6702, Name: "Silver of Caria Spectral Steed Attire", ItemID: 0x401EAA0A},
	{FlagID: 6703, Name: "Funereal Night Spectral Steed Attire", ItemID: 0x401EAA14},
}

// SpectralSteedAttireDefaultFlag is the appearance the game selects when no
// attire is worn.
const SpectralSteedAttireDefaultFlag = uint32(6700)

// FindSpectralSteedAttire returns the appearance owning flagID.
func FindSpectralSteedAttire(flagID uint32) (SpectralSteedAttire, bool) {
	for _, a := range SpectralSteedAttires {
		if a.FlagID == flagID {
			return a, true
		}
	}
	return SpectralSteedAttire{}, false
}
