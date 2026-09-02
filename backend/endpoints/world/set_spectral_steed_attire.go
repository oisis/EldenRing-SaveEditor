/*
Endpoint: SetSpectralSteedAttire
EndpointID: set_spectral_steed_attire
Purpose: Activates one Spectral Steed Attire appearance of Torrent.
How it works: The handler resolves the shared appearance table against GameCatalog, maps the public attire key onto its event flag and required item, and delegates one atomic event-flag mutation to SaveEngine, which rejects an appearance whose item is not in Inventory before it moves a byte.
Supported resource types: ItemDocument: Spectral Steed Attire.
Input variables: saveSessionID, characterID, attireKey, expectedRevision.
GameCatalog variables read: resource kind and key plus item family and gameID of the three Regulation 1.17 attire items.
Save variables processed: the four mutually exclusive Spectral Steed Attire event flags and, read-only, the positive-quantity Inventory record of the selected appearance. No item is created, moved or removed.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetSpectralSteedAttireEndpointID is the stable backend identifier of SetSpectralSteedAttire.
const SetSpectralSteedAttireEndpointID = "set_spectral_steed_attire"

// SetSpectralSteedAttireDefinition describes the public mutation contract.
var SetSpectralSteedAttireDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSpectralSteedAttire",
	ID:                         SetSpectralSteedAttireEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Spectral Steed Attire",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "attireKey", "expectedRevision"},
	Description:                "Activates one Spectral Steed Attire appearance of Torrent.",
})

// SetSpectralSteedAttireResult reports the committed appearance in public terms
// without exposing the event flag behind it.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetSpectralSteedAttireResult struct {
	saveengine.MutationReceipt
	CharacterID int    `json:"characterID"`
	AttireKey   string `json:"attireKey"`
}

// SetSpectralSteedAttire activates exactly one appearance of the shared table.
//
// attireKey accepts only one of the four public keys. The default appearance
// needs no item; the other three require their attire item in Inventory, and a
// missing item rejects the call before any mutation and without advancing the
// revision. This endpoint never adds the item: it is added beforehand through
// AddItemToInventory by the item's exact GameCatalog resource key.
//
// The mutation clears all four appearance flags and sets the selected one, so it
// also resolves a legacy save whose flags are all cleared and a conflicting save
// with more than one set.
func SetSpectralSteedAttire(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	attireKey string,
	expectedRevision string,
) (SetSpectralSteedAttireResult, error) {
	if engine == nil {
		return SetSpectralSteedAttireResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetSpectralSteedAttireResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogSpectralSteedAttires(gameCatalog)
	if err != nil {
		return SetSpectralSteedAttireResult{}, err
	}
	selected, found := declaredSpectralSteedAttireByKey(declared, attireKey)
	if !found {
		return SetSpectralSteedAttireResult{}, fmt.Errorf(
			"attireKey must be one of %s; got %q",
			strings.Join(spectralSteedAttireKeys(declared), ", "), attireKey)
	}

	mutation, err := engine.SetSpectralSteedAttire(
		saveSessionID,
		characterID,
		spectralSteedAttireStates(declared),
		selected.eventFlagID,
		expectedRevision,
	)
	if err != nil {
		return SetSpectralSteedAttireResult{}, err
	}
	return SetSpectralSteedAttireResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		AttireKey:       selected.entry.AttireKey,
	}, nil
}

func declaredSpectralSteedAttireByKey(
	declared []declaredSpectralSteedAttire, attireKey string,
) (declaredSpectralSteedAttire, bool) {
	for _, attire := range declared {
		if attire.entry.AttireKey == attireKey {
			return attire, true
		}
	}
	return declaredSpectralSteedAttire{}, false
}

func spectralSteedAttireKeys(declared []declaredSpectralSteedAttire) []string {
	keys := make([]string, 0, len(declared))
	for _, attire := range declared {
		keys = append(keys, strconv.Quote(attire.entry.AttireKey))
	}
	return keys
}
