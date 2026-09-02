package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// UserData10 layout of the 15 Mirror Favorites preset slots, shared by PC and
// PS4. Presets are stored globally in UserData10 and are shared across all
// character slots of the save session.
const (
	favoriteSlotCount   = 15
	favoriteSlotSize    = 0x130
	favoriteBaseOffset  = 0x154
	favoriteMagicOffset = 0x18
	favoriteMagic       = "FACE"
)

// FavoritePreset represents the occupancy state of one Mirror Favorites slot.
type FavoritePreset struct {
	FavoriteSlotID int  `json:"favoriteSlotID"`
	Active         bool `json:"active"`
}

// FavoritePresetsState is the result of GetFavoritePresets in SaveEngine.
type FavoritePresetsState struct {
	SaveSessionID string           `json:"saveSessionID"`
	Presets       []FavoritePreset `json:"presets"`
}

// GetFavoritePresets returns the occupancy state of Mirror Favorites preset slots
// from the session's private snapshot.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed.
//
// favoriteSlotID:
// - nil returns all 15 slots in order 0..14;
// - a value in 0..14 returns exactly that one slot;
// - any other value is rejected.
func (engine *Engine) GetFavoritePresets(
	saveSessionID string,
	favoriteSlotID *int,
) (FavoritePresetsState, error) {
	if saveSessionID == "" {
		return FavoritePresetsState{}, apperror.MissingField("saveSessionID")
	}
	if favoriteSlotID != nil && (*favoriteSlotID < 0 || *favoriteSlotID >= favoriteSlotCount) {
		return FavoritePresetsState{}, fmt.Errorf(
			"favoriteSlotID %d is outside the range 0..%d", *favoriteSlotID, favoriteSlotCount-1)
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return FavoritePresetsState{}, apperror.UnknownSaveSession(saveSessionID)
	}

	base := userData10Base(loaded.session.platform)

	startSlot := 0
	endSlot := favoriteSlotCount
	if favoriteSlotID != nil {
		startSlot = *favoriteSlotID
		endSlot = *favoriteSlotID + 1
	}

	presets := make([]FavoritePreset, 0, endSlot-startSlot)
	for slot := startSlot; slot < endSlot; slot++ {
		active, err := readFavoriteSlotActive(loaded.snapshot, base, slot)
		if err != nil {
			return FavoritePresetsState{}, err
		}
		presets = append(presets, FavoritePreset{
			FavoriteSlotID: slot,
			Active:         active,
		})
	}

	return FavoritePresetsState{
		SaveSessionID: saveSessionID,
		Presets:       presets,
	}, nil
}

func favoriteSlotOffset(base int64, slot int) int64 {
	return base + favoriteBaseOffset + int64(slot)*favoriteSlotSize
}

func readFavoriteSlotActive(snapshot *codec, base int64, slot int) (bool, error) {
	slotAt := favoriteSlotOffset(base, slot)
	if !snapshot.covers(slotAt, favoriteSlotSize) {
		return false, fmt.Errorf("favorite preset slot %d lies outside UserData10 bounds", slot)
	}

	magic, err := snapshot.readAt(slotAt+favoriteMagicOffset, len(favoriteMagic))
	if err != nil {
		return false, fmt.Errorf("cannot read magic of favorite preset slot %d: %w", slot, err)
	}
	return string(magic) == favoriteMagic, nil
}
