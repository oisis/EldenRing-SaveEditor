/*
Endpoint: GetSpectralSteedAttires
EndpointID: get_spectral_steed_attires
Purpose: Returns the four Spectral Steed Attire appearances of Torrent with their ownership and the active selection.
How it works: The handler resolves the four appearances of the shared appearance table against GameCatalog, reads their mutually exclusive event flags and the positive-quantity Inventory records of the three attire items, and reports which appearance is active or why that cannot be decided.
Supported resource types: ItemDocument: Spectral Steed Attire.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus item family, gameID and presentation iconPath of the three Regulation 1.17 attire items.
Save variables read: the character activity flag, the four Spectral Steed Attire event flag bits and positive-quantity goods records in common and key InventoryHeld; the getter writes nothing.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// requireSameSaveRevision prevents a world projection assembled from more than
// one SaveEngine read from mixing two committed session snapshots. Revisions
// are opaque and therefore compared exactly, never parsed or ordered.
func requireSameSaveRevision(expected string, actual string) error {
	if actual == expected {
		return nil
	}
	return fmt.Errorf(
		"save revision changed during read: first read returned %q, later read returned %q",
		expected,
		actual,
	)
}

// GetSpectralSteedAttiresEndpointID is the stable backend identifier of GetSpectralSteedAttires.
const GetSpectralSteedAttiresEndpointID = "get_spectral_steed_attires"

// GetSpectralSteedAttiresDefinition describes the public getter contract.
var GetSpectralSteedAttiresDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSpectralSteedAttires",
	ID:                         GetSpectralSteedAttiresEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Spectral Steed Attire",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns the four Spectral Steed Attire appearances of Torrent with their ownership and the active selection.",
})

// Spectral Steed Attire resolution states. The getter never repairs a save, so
// it reports what it found instead of guessing an active appearance.
const (
	// SpectralSteedAttireStatusResolved — exactly one appearance flag is set and
	// activeAttireKey names it.
	SpectralSteedAttireStatusResolved = "resolved"
	// SpectralSteedAttireStatusLegacy — all four flags are cleared, which is the
	// normal state of a save that predates Regulation 1.17. The active appearance
	// stays unknown: a cleared set is deliberately not read as "default".
	SpectralSteedAttireStatusLegacy = "legacy"
	// SpectralSteedAttireStatusConflict — two or more flags are set at once.
	SpectralSteedAttireStatusConflict = "conflict"
)

// Public appearance keys. They are the stable identity of the three endpoints of
// this feature; the event flags behind them stay a save-format detail.
const (
	SpectralSteedAttireKeyDefault       = "default"
	SpectralSteedAttireKeyTreeSentinel  = "tree_sentinel"
	SpectralSteedAttireKeySilverCaria   = "silver_of_caria"
	SpectralSteedAttireKeyFunerealNight = "funereal_night"
)

// spectralSteedAttireDefinition is one row of the appearance table below.
//
// resourceKey is empty for the default appearance, which the game offers without
// an item and therefore without an icon. Name is the appearance name shown to the
// user, which is deliberately not the catalog item name: the item is the garb, the
// appearance is what Torrent wears.
type spectralSteedAttireDefinition struct {
	key         string
	name        string
	eventFlagID uint32
	resourceKey string
}

// spectralSteedAttireDefinitions is the single source of truth of this feature.
// GetSpectralSteedAttires, SetSpectralSteedAttire and LockAllSpectralSteedAttires
// all read this one table, so the appearance order, the public keys, the names,
// the event flags and the required items cannot drift apart between them.
//
// Confirmed contract: event flags 6700-6703 are mutually exclusive and the game
// sets exactly one of them once the Spectral Steed Attire menu has been opened.
// The three attire items are Regulation 1.17 goods resolved by their exact
// GameCatalog resource key; they carry no catalog unlock declaration, so the
// flag they select is owned here and nowhere else.
var spectralSteedAttireDefinitions = []spectralSteedAttireDefinition{
	{key: SpectralSteedAttireKeyDefault, name: "Default Appearance", eventFlagID: 6700},
	{key: SpectralSteedAttireKeyTreeSentinel, name: "Tree Sentinel Spectral Steed Attire",
		eventFlagID: 6701, resourceKey: "401EAA00"},
	{key: SpectralSteedAttireKeySilverCaria, name: "Silver of Caria Spectral Steed Attire",
		eventFlagID: 6702, resourceKey: "401EAA0A"},
	{key: SpectralSteedAttireKeyFunerealNight, name: "Funereal Night Spectral Steed Attire",
		eventFlagID: 6703, resourceKey: "401EAA14"},
}

// SpectralSteedAttireEntry is one appearance offered to the user. An appearance
// without a required item — the default one — carries an empty resource
// reference and an empty icon path; nothing is invented for it.
type SpectralSteedAttireEntry struct {
	AttireKey            string              `json:"attireKey"`
	Name                 string              `json:"name"`
	Owned                bool                `json:"owned"`
	RequiredResourceKind schema.ResourceKind `json:"requiredResourceKind"`
	RequiredResourceKey  string              `json:"requiredResourceKey"`
	IconPath             string              `json:"iconPath"`
}

// GetSpectralSteedAttiresResult is the read-only view of the four appearances.
// ActiveAttireKey is filled only when Status is resolved.
type GetSpectralSteedAttiresResult struct {
	SaveSessionID   string                     `json:"saveSessionID"`
	SaveRevision    string                     `json:"saveRevision"`
	CharacterID     int                        `json:"characterID"`
	Active          bool                       `json:"active"`
	Status          string                     `json:"status"`
	ActiveAttireKey string                     `json:"activeAttireKey"`
	Attires         []SpectralSteedAttireEntry `json:"attires"`
}

// GetSpectralSteedAttires reports the four appearances in menu order together
// with the appearance the save has selected.
//
// The default appearance is always owned; the other three are owned only while
// their item has a positive Inventory record. Storage does not count and a set
// event flag is never read as proof of ownership, so an appearance whose item was
// dropped is reported as not owned even while it is worn.
//
// The getter neither mutates nor repairs. An unreadable event flag region is an
// error, never a legacy state.
func GetSpectralSteedAttires(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetSpectralSteedAttiresResult, error) {
	if engine == nil {
		return GetSpectralSteedAttiresResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetSpectralSteedAttiresResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogSpectralSteedAttires(gameCatalog)
	if err != nil {
		return GetSpectralSteedAttiresResult{}, err
	}
	eventFlagIDs := make([]uint32, 0, len(declared))
	gameIDs := make([]uint32, 0, len(declared))
	for _, attire := range declared {
		eventFlagIDs = append(eventFlagIDs, attire.eventFlagID)
		if attire.gameID != 0 {
			gameIDs = append(gameIDs, attire.gameID)
		}
	}

	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetSpectralSteedAttiresResult{}, err
	}
	owned := map[uint32]bool{}
	if flags.Active {
		presence, err := engine.GetInventoryGoodsPresence(saveSessionID, characterID, gameIDs)
		if err != nil {
			return GetSpectralSteedAttiresResult{}, err
		}
		if err := requireSameSaveRevision(flags.SaveRevision, presence.SaveRevision); err != nil {
			return GetSpectralSteedAttiresResult{}, err
		}
		owned = presence.Presence
	}

	result := GetSpectralSteedAttiresResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		Status:        SpectralSteedAttireStatusLegacy,
		Attires:       make([]SpectralSteedAttireEntry, 0, len(declared)),
	}
	setCount := 0
	for _, attire := range declared {
		entry := attire.entry
		entry.Owned = attire.gameID == 0 || owned[attire.gameID]
		result.Attires = append(result.Attires, entry)
		if flags.Flags[attire.eventFlagID] {
			setCount++
			result.ActiveAttireKey = entry.AttireKey
		}
	}
	switch {
	case setCount == 1:
		result.Status = SpectralSteedAttireStatusResolved
	case setCount > 1:
		result.Status = SpectralSteedAttireStatusConflict
		result.ActiveAttireKey = ""
	default:
		// No flag is set. That is the normal state of a save written before
		// Regulation 1.17 and of an inactive slot, whose bitfield is never read.
		result.ActiveAttireKey = ""
	}
	return result, nil
}

// declaredSpectralSteedAttire joins one row of the appearance table with the
// catalog data behind it.
type declaredSpectralSteedAttire struct {
	entry       SpectralSteedAttireEntry
	eventFlagID uint32
	gameID      uint32
}

// catalogSpectralSteedAttires resolves the appearance table against GameCatalog.
// It is shared by the getter and both mutations, so all three reject the same
// incomplete catalog data instead of each deciding for itself.
//
// Every attire is resolved by its exact resource key, which keeps the three
// Regulation 1.17 items reachable while they stay out of the general item
// catalog. Unknown, non-goods or duplicated data fails closed.
func catalogSpectralSteedAttires(
	gameCatalog *gamecatalog.Catalog,
) ([]declaredSpectralSteedAttire, error) {
	declared := make([]declaredSpectralSteedAttire, 0, len(spectralSteedAttireDefinitions))
	gameIDOwners := make(map[uint32]string, len(spectralSteedAttireDefinitions))
	for _, definition := range spectralSteedAttireDefinitions {
		attire := declaredSpectralSteedAttire{
			entry:       SpectralSteedAttireEntry{AttireKey: definition.key, Name: definition.name},
			eventFlagID: definition.eventFlagID,
		}
		if definition.resourceKey == "" {
			declared = append(declared, attire)
			continue
		}

		resource, err := gameCatalog.ResourceByKindAndKey(
			schema.ResourceKindItem, definition.resourceKey)
		if err != nil {
			return nil, fmt.Errorf("Spectral Steed Attire %q: %w", definition.key, err)
		}
		if resource.Item == nil {
			return nil, fmt.Errorf(
				"Spectral Steed Attire %q resolves to resource %q without an item document",
				definition.key, definition.resourceKey)
		}
		if !resource.Item.Family.Known || resource.Item.Family.Value != schema.ItemFamilyGoods {
			return nil, fmt.Errorf(
				"Spectral Steed Attire %q must have known item family %q",
				definition.key, schema.ItemFamilyGoods)
		}
		if !resource.Item.GameID.Known || resource.Item.GameID.Value&0xF0000000 != 0x40000000 {
			return nil, fmt.Errorf(
				"Spectral Steed Attire %q has no valid goods game ID", definition.key)
		}
		if owner, duplicate := gameIDOwners[resource.Item.GameID.Value]; duplicate {
			return nil, fmt.Errorf(
				"Spectral Steed Attires %q and %q both declare game ID 0x%08X",
				owner, definition.key, resource.Item.GameID.Value)
		}
		gameIDOwners[resource.Item.GameID.Value] = definition.key

		attire.gameID = resource.Item.GameID.Value
		attire.entry.RequiredResourceKind = resource.Kind
		attire.entry.RequiredResourceKey = resource.Key
		if resource.Item.Presentation.IconPath.Known {
			attire.entry.IconPath = resource.Item.Presentation.IconPath.Value
		}
		declared = append(declared, attire)
	}
	return declared, nil
}

// spectralSteedAttireStates projects the resolved table onto the save-side set
// SaveEngine validates and mutates.
func spectralSteedAttireStates(
	declared []declaredSpectralSteedAttire,
) []saveengine.SpectralSteedAttire {
	states := make([]saveengine.SpectralSteedAttire, 0, len(declared))
	for _, attire := range declared {
		states = append(states, saveengine.SpectralSteedAttire{
			EventFlagID: attire.eventFlagID,
			GameID:      attire.gameID,
		})
	}
	return states
}
