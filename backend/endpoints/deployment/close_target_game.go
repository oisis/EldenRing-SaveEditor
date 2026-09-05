/*
Endpoint: CloseTargetGame
EndpointID: close_target_game
Purpose: Runs the configured stop command of one target as an explicit user action.
How it works: The runtime handler executes exactly the command line the user configured and returns its real outcome, including the case where no process was found.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented
*/
package deployment

import (
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// CloseTargetGameEndpointID is the stable backend identifier of CloseTargetGame.
const CloseTargetGameEndpointID = "close_target_game"

// CloseTargetGameDefinition describes the public mutation contract.
var CloseTargetGameDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CloseTargetGame",
	ID:                         CloseTargetGameEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Runs the configured stop command of one target as an explicit user action.",
})

// CloseTargetGame runs the configured stop command.
func CloseTargetGame(
	ctx context.Context, service *deployment.Service, targetID string,
) (CommandOutcome, error) {
	if service == nil {
		return CommandOutcome{}, errors.New("deployment service is required")
	}
	return service.CloseGame(ctx, targetID)
}
