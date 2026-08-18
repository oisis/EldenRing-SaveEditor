/*
Endpoint: GetEquippedSpells
EndpointID: get_equipped_spells
Purpose: Returns spells in memory slots together with used and available Memory Slots capacity.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session, and resolves every occupied raw MagicParam ID through GameCatalog. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: ItemDocument: Spell.
Input variables: saveSessionID, characterID.
GameCatalog variables read: for every occupied spell the resource key, the item family, the presentation name and the spell Memory Slots cost of the ItemDocument whose game ID is the raw MagicParam ID with the spell item-family prefix.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the fourteen EquippedSpells records of its slot data plus the Memory Stone stack and the unlocked talisman fields the active capacity is derived from; the getter is non-mutating and writes nothing.
Implementation status: implemented
*/
package equipment

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetEquippedSpellsEndpointID is the stable backend identifier of GetEquippedSpells.
const GetEquippedSpellsEndpointID = "get_equipped_spells"

// GetEquippedSpellsDefinition describes the public getter contract.
var GetEquippedSpellsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetEquippedSpells",
	ID:                         GetEquippedSpellsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Spell",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns spells in memory slots together with used and available Memory Slots capacity.",
})

// equippedSpellRawIDLimit is the first value a raw MagicParam ID may no longer
// reach: a stored identifier that already carries family bits is not a raw ID
// and is never prefixed a second time. The item-family prefix itself lives in
// GameCatalog as gamecatalog.EquippedSpellGameIDPrefix.
const equippedSpellRawIDLimit uint32 = 0x10000000

// equippedSpellEmptyID is the raw identifier the game stores in an empty record.
// It is preserved in the result, and it is the one occupied-slot rule that never
// applies: an empty record resolves nothing.
const equippedSpellEmptyID uint32 = 0xFFFFFFFF

// EquippedSpellSlot is one physical EquippedSpells record. RawMagicParamID is
// the identifier exactly as the save stores it, including the empty sentinel.
// The three resolved fields come from GameCatalog and are filled for an occupied
// record only, so an empty record keeps its sentinel and exposes no resource
// data at all.
type EquippedSpellSlot struct {
	RawMagicParamID uint32 `json:"rawMagicParamID"`
	ResourceKey     string `json:"resourceKey"`
	Name            string `json:"name"`
	MemorySlots     int    `json:"memorySlots"`
}

// GetEquippedSpellsResult is the typed result of GetEquippedSpells: the twelve
// public playable spell memory slots in stored order, the Memory Slots the occupied ones consume,
// and the Memory Slots the character may fill.
//
// An inactive slot — including a residual one — reports Active false, twelve
// zero-value records and both counts zero.
type GetEquippedSpellsResult struct {
	SaveSessionID        string              `json:"saveSessionID"`
	CharacterID          int                 `json:"characterID"`
	Active               bool                `json:"active"`
	Spells               []EquippedSpellSlot `json:"spells"`
	UsedMemorySlots      int                 `json:"usedMemorySlots"`
	AvailableMemorySlots int                 `json:"availableMemorySlots"`
}

// GetEquippedSpells returns the equipped spells of one character slot of an
// existing save session, with every occupied record resolved through
// GameCatalog.
//
// The endpoint is thin: it rejects a missing engine and a missing catalog, and
// delegates everything else. Validating saveSessionID and characterID, reading
// the snapshot, validating the native record pairs and computing the available
// capacity belong to SaveEngine; naming a spell and stating its cost belong to
// GameCatalog. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file and returns
// no raw save byte.
//
// An occupied record whose raw identifier does not resolve to a known Spell with
// a known name and a known Memory Slots cost is a fail-closed error: no name,
// key or cost is invented for it.
func GetEquippedSpells(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetEquippedSpellsResult, error) {
	if engine == nil {
		return GetEquippedSpellsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetEquippedSpellsResult{}, errors.New("game catalog is not available")
	}

	stored, err := engine.GetEquippedSpells(saveSessionID, characterID)
	if err != nil {
		return GetEquippedSpellsResult{}, err
	}

	const publicSpellSlotCount = 12
	result := GetEquippedSpellsResult{
		SaveSessionID: stored.SaveSessionID,
		CharacterID:   stored.CharacterID,
		Active:        stored.Active,
		Spells:        make([]EquippedSpellSlot, publicSpellSlotCount),
	}
	if !stored.Active {
		// An inactive slot carries no spell state at all, so nothing is resolved
		// and both counts stay zero.
		return result, nil
	}

	for index, raw := range stored.Spells[:publicSpellSlotCount] {
		result.Spells[index] = EquippedSpellSlot{RawMagicParamID: raw}
		if raw == equippedSpellEmptyID {
			continue
		}
		resolved, err := resolveEquippedSpell(gameCatalog, raw)
		if err != nil {
			return GetEquippedSpellsResult{}, fmt.Errorf("spell slot %d: %w", index, err)
		}
		resolved.RawMagicParamID = raw
		result.Spells[index] = resolved
		result.UsedMemorySlots += resolved.MemorySlots
	}
	result.AvailableMemorySlots = stored.AvailableMemorySlots
	return result, nil
}

// resolveEquippedSpell turns one raw MagicParam ID into the resource key, name
// and Memory Slots cost GameCatalog stores for it. The raw format is validated
// before the item-family prefix is applied, so a stored value that is not a raw
// MagicParam ID is rejected instead of being converted into a different item.
func resolveEquippedSpell(gameCatalog *gamecatalog.Catalog, raw uint32) (EquippedSpellSlot, error) {
	if raw == 0 || raw >= equippedSpellRawIDLimit {
		return EquippedSpellSlot{}, fmt.Errorf("0x%08X is not a raw MagicParam ID", raw)
	}

	gameID := gamecatalog.EquippedSpellGameIDPrefix | raw
	resource, exists := gameCatalog.ItemByGameID(gameID)
	if !exists || resource.Item == nil {
		return EquippedSpellSlot{}, fmt.Errorf("game ID 0x%08X is not a known item", gameID)
	}
	item := resource.Item
	if !item.Family.Known || item.Family.Value != schema.ItemFamilySpell {
		return EquippedSpellSlot{}, fmt.Errorf("game ID 0x%08X is not a spell", gameID)
	}
	if !item.Presentation.Name.Known {
		return EquippedSpellSlot{}, fmt.Errorf("spell 0x%08X has no known name", gameID)
	}
	if item.Spell == nil || !item.Spell.MemorySlots.Known {
		return EquippedSpellSlot{}, fmt.Errorf("spell 0x%08X has no known memory slots", gameID)
	}
	return EquippedSpellSlot{
		ResourceKey: resource.Key,
		Name:        item.Presentation.Name.Value,
		MemorySlots: int(item.Spell.MemorySlots.Value),
	}, nil
}
