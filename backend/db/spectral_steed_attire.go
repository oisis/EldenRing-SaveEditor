package db

import "github.com/oisis/EldenRing-SaveForge/backend/db/data"

// SpectralSteedAttireEntry is one Torrent appearance offered to the user.
// IconPath is read from the item database so the icon has a single owner;
// the default appearance has no item and therefore no icon.
type SpectralSteedAttireEntry struct {
	ID       uint32 `json:"id"` // event flag ID (6700-6703)
	Name     string `json:"name"`
	ItemID   uint32 `json:"itemId"` // 0 for the default appearance
	IconPath string `json:"iconPath"`
	Owned    bool   `json:"owned"` // required item present in inventory
}

// Spectral Steed Attire resolution states. The getter never repairs a save, so
// it reports what it found instead of guessing an active appearance.
const (
	// SpectralSteedAttireResolved — exactly one of 6700-6703 is set.
	SpectralSteedAttireResolved = "resolved"
	// SpectralSteedAttireLegacy — all four flags are cleared, which is the
	// normal state of a save that predates Regulation 1.17.
	SpectralSteedAttireLegacy = "legacy"
	// SpectralSteedAttireConflict — two or more flags are set at once.
	SpectralSteedAttireConflict = "conflict"
)

// SpectralSteedAttireState is the read-only view of the four appearances.
// ActiveID is meaningful only when Status is SpectralSteedAttireResolved.
type SpectralSteedAttireState struct {
	Entries  []SpectralSteedAttireEntry `json:"entries"`
	ActiveID uint32                     `json:"activeId"`
	Status   string                     `json:"status"`
}

// GetAllSpectralSteedAttires returns the four appearances with icons resolved
// and no save-derived state applied.
func GetAllSpectralSteedAttires() []SpectralSteedAttireEntry {
	entries := make([]SpectralSteedAttireEntry, 0, len(data.SpectralSteedAttires))
	for _, a := range data.SpectralSteedAttires {
		entry := SpectralSteedAttireEntry{ID: a.FlagID, Name: a.Name, ItemID: a.ItemID}
		if a.ItemID != 0 {
			entry.IconPath = GetItemData(a.ItemID).IconPath
		}
		entries = append(entries, entry)
	}
	return entries
}
