/*
Endpoint: GetWorldMutationCapabilities
EndpointID: get_world_mutation_capabilities
Purpose: Returns the supported World mutations with the backend risk contract of each one.
How it works: The handler walks its own fixed list of the fifteen World mutation EndpointIDs and asks SaveEngine to describe each of them. Risk and risk reason are the values the operation history uses, so no second risk table exists; an operation kind SaveEngine does not know is rejected instead of being published with a default risk.
Supported resource types: none; the contract describes operations, not catalog resources.
Input variables: none. The contract is static and names neither a session nor a character.
GameCatalog variables read: none.
Save variables read: none; the endpoint opens no session and reads no save data.
Implementation status: implemented
*/
package world

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetWorldMutationCapabilitiesEndpointID is the stable backend identifier of GetWorldMutationCapabilities.
const GetWorldMutationCapabilitiesEndpointID = "get_world_mutation_capabilities"

// GetWorldMutationCapabilitiesDefinition describes the public getter contract.
var GetWorldMutationCapabilitiesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetWorldMutationCapabilities",
	ID:                         GetWorldMutationCapabilitiesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "none",
	SupportedResourceVariables: nil,
	Description:                "Returns the supported World mutations with the backend risk contract of each one.",
})

// WorldMutationCapability is one supported World mutation. Its presence is what
// declares the operation supported; a frontend therefore never carries its own
// list of World writers, and never its own risk level either.
type WorldMutationCapability struct {
	OperationKind string                   `json:"operationKind"`
	Risk          saveengine.OperationRisk `json:"risk"`
	RiskReason    string                   `json:"riskReason"`
	SupportsBulk  bool                     `json:"supportsBulk"`
}

// GetWorldMutationCapabilitiesResult is the complete, deterministic World
// mutation contract.
type GetWorldMutationCapabilitiesResult struct {
	Capabilities []WorldMutationCapability `json:"capabilities"`
}

// worldMutationEndpointIDs is the authored, alphabetically ordered list of the
// World mutations this package implements. It holds the EndpointID constants of
// those endpoints, which are the SaveEngine operation kinds they commit under,
// so the capability list cannot drift from the endpoints it describes.
var worldMutationEndpointIDs = []string{
	LockAllSpectralSteedAttiresEndpointID,
	SetBellBearingUnlockedEndpointID,
	SetBossDefeatedEndpointID,
	SetColosseumUnlockedEndpointID,
	SetCookbookUnlockedEndpointID,
	SetFogOfWarRemovedEndpointID,
	SetGestureUnlockedEndpointID,
	SetGraceVisitedEndpointID,
	SetMapRegionRevealedEndpointID,
	SetQuestStepEndpointID,
	SetRegionUnlockedEndpointID,
	SetSpectralSteedAttireEndpointID,
	SetSummoningPoolActivatedEndpointID,
	SetTutorialUnlockedEndpointID,
	SetWhetbladeUnlockedEndpointID,
}

// bulkWorldMutationOpKind is the one World mutation that changes a whole set
// in a single atomic commit. Every other capability is a single-target write, so
// a caller that wants many of them performs many mutations and reviews each one.
const bulkWorldMutationOpKind = LockAllSpectralSteedAttiresEndpointID

// GetWorldMutationCapabilities returns the supported World mutations together
// with the risk the backend attaches to each of them.
//
// It takes no session and no character: which World mutations exist and what
// risk they carry is a property of the build, not of a save. The risk and its
// reason are read from SaveEngine, the same source Review Changes presents, so
// the contract has exactly one owner.
func GetWorldMutationCapabilities() (GetWorldMutationCapabilitiesResult, error) {
	capabilities := make([]WorldMutationCapability, 0, len(worldMutationEndpointIDs))
	for _, operationKind := range worldMutationEndpointIDs {
		described, err := saveengine.DescribeMutationOperation(operationKind)
		if err != nil {
			return GetWorldMutationCapabilitiesResult{}, fmt.Errorf(
				"world mutation %q: %w", operationKind, err)
		}
		capabilities = append(capabilities, WorldMutationCapability{
			OperationKind: described.OperationKind,
			Risk:          described.Risk,
			RiskReason:    described.RiskReason,
			SupportsBulk:  operationKind == bulkWorldMutationOpKind,
		})
	}
	return GetWorldMutationCapabilitiesResult{Capabilities: capabilities}, nil
}
