/*
Endpoint: ApplyRepairs
EndpointID: apply_repairs
Purpose: Re-derives and atomically executes the executable actions of a selected GetRepairPlan result.
How it works: The runtime handler re-derives the selected plan from issueIDs at expectedRevision, verifies its planToken and delegates its executable actions as one SaveEngine transaction. Rejected findings are returned but never mutate the save.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, issueIDs, planToken, expectedRevision.
GameCatalog variables read: the same container limits and stack rules GetRepairPlan uses to derive its executable actions.
Save variables processed: the physical records and statistics named by the freshly derived actions; SaveEngine validates targets under one lock and commits all writes or none.
Implementation status: implemented
*/
package diagnostics

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// ApplyRepairsEndpointID is the stable backend identifier of ApplyRepairs.
const ApplyRepairsEndpointID = "apply_repairs"

// ApplyRepairsDefinition describes the public mutation contract.
var ApplyRepairsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyRepairs",
	ID:                         ApplyRepairsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "issueIDs", "planToken", "expectedRevision"},
	Description:                "Re-derives and atomically executes the executable actions of a selected GetRepairPlan result.",
})

// ApplyRepairsResult is the receipt of applying the executable actions selected
// by issueIDs. Rejected is echoed from the freshly derived plan so a caller can
// never mistake an unrepairable finding for a successful mutation.
type ApplyRepairsResult struct {
	SaveSessionID string            `json:"saveSessionID"`
	SaveRevision  string            `json:"saveRevision"`
	CharacterID   int               `json:"characterID"`
	Applied       bool              `json:"applied"`
	Actions       []RepairAction    `json:"actions"`
	Rejected      []RepairRejection `json:"rejected"`
}

// ApplyRepairs re-derives the selected repair plan at expectedRevision and
// accepts planToken only when it seals the same executable actions. issueIDs
// are required because a digest can verify a plan but cannot address one.
func ApplyRepairs(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	issueIDs []string,
	planToken string,
	expectedRevision string,
) (ApplyRepairsResult, error) {
	if engine == nil {
		return ApplyRepairsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return ApplyRepairsResult{}, errors.New("game catalog is not available")
	}
	if !saveengine.IsCanonicalRevision(expectedRevision) {
		return ApplyRepairsResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	plan, err := GetRepairPlan(
		engine, gameCatalog, saveSessionID, characterID, expectedRevision, issueIDs)
	if err != nil {
		return ApplyRepairsResult{}, err
	}
	if plan.PlanToken != planToken {
		return ApplyRepairsResult{}, errors.New("planToken does not match the executable repair actions of this save revision")
	}

	actions := make([]saveengine.RepairAction, len(plan.Actions))
	for index, action := range plan.Actions {
		actions[index] = saveengine.RepairAction{
			Operation:   action.Operation,
			OwnedItemID: action.OwnedItemID,
			TargetValue: action.TargetValue,
			Attributes:  action.Attributes,
		}
	}
	mutation, err := engine.ApplyRepairPlan(saveSessionID, characterID, actions, expectedRevision)
	if err != nil {
		return ApplyRepairsResult{}, err
	}
	return ApplyRepairsResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		Applied:       mutation.Applied,
		Actions:       plan.Actions,
		Rejected:      plan.Rejected,
	}, nil
}
