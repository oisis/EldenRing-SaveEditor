/*
Endpoint: GetPhysickMixture
EndpointID: get_physick_mixture
Purpose: Returns both entries of the current Flask of Wondrous Physick mixture.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none; this stage returns raw state and resolves no ItemDocument.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the two raw Crystal Tear identifiers at the start of the EquipPhysicsData block of its slot data; the getter is non-mutating and computes no value.
Implementation status: implemented
*/
package equipment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetPhysickMixtureEndpointID is the stable backend identifier of GetPhysickMixture.
const GetPhysickMixtureEndpointID = "get_physick_mixture"

// GetPhysickMixtureDefinition describes the public getter contract.
var GetPhysickMixtureDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetPhysickMixture",
	ID:                         GetPhysickMixtureEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns both entries of the current Flask of Wondrous Physick mixture.",
})

// GetPhysickMixtureResult is the typed result of GetPhysickMixture. The shape is
// owned by SaveEngine, so the endpoint neither reshapes nor duplicates it. Both
// values stay exactly as stored in the save.
type GetPhysickMixtureResult = saveengine.CharacterPhysickMixture

// GetPhysickMixture returns the raw Physick mixture stored in one character slot
// of an existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file, reads no
// GameCatalog and returns no raw save byte.
func GetPhysickMixture(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
) (GetPhysickMixtureResult, error) {
	if engine == nil {
		return GetPhysickMixtureResult{}, errors.New("save engine is not available")
	}
	return engine.GetPhysickMixture(saveSessionID, characterID)
}
