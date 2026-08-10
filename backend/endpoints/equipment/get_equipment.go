/*
Endpoint: GetEquipment
EndpointID: get_equipment
Purpose: Returns the 22 raw ChrAsmEquipment fields of one character slot without resolving them through GameCatalog.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none; this stage returns raw state and resolves no ItemDocument.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the 22 raw equipped-armaments fields of its slot data; the getter is non-mutating and computes no value.
Implementation status: implemented.
*/
package equipment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetEquipmentEndpointID is the stable backend identifier of GetEquipment.
const GetEquipmentEndpointID = "get_equipment"

// GetEquipmentDefinition describes the public getter contract.
var GetEquipmentDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetEquipment",
	ID:                         GetEquipmentEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns the 22 raw ChrAsmEquipment fields of one character slot without resolving them through GameCatalog.",
})

// GetEquipmentResult is the typed result of GetEquipment. The shape is owned by
// SaveEngine, so the endpoint neither reshapes nor duplicates it. Every value
// stays exactly as stored in the save.
type GetEquipmentResult = saveengine.CharacterEquipment

// GetEquipment returns the raw equipped state stored in one character slot of an
// existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file, reads no
// GameCatalog and returns no raw save byte.
func GetEquipment(engine *saveengine.Engine, saveSessionID string, characterID int) (GetEquipmentResult, error) {
	if engine == nil {
		return GetEquipmentResult{}, errors.New("save engine is not available")
	}
	return engine.GetEquipment(saveSessionID, characterID)
}
