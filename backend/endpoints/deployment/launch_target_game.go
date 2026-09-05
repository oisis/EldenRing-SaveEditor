/*
Endpoint: LaunchTargetGame
EndpointID: launch_target_game
Purpose: Runs the configured start command of one target as an explicit user action.
How it works: The runtime handler executes exactly the command line the user configured and returns its real outcome.
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

// LaunchTargetGameEndpointID is the stable backend identifier of LaunchTargetGame.
const LaunchTargetGameEndpointID = "launch_target_game"

// LaunchTargetGameDefinition describes the public mutation contract.
var LaunchTargetGameDefinition = contract.MustDefine(contract.Definition{
	Name:                       "LaunchTargetGame",
	ID:                         LaunchTargetGameEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Runs the configured start command of one target as an explicit user action.",
})

// CommandOutcome is the typed result of Launch and Close Game.
type CommandOutcome = deployment.CommandOutcome

// LaunchTargetGame runs the configured start command.
func LaunchTargetGame(
	ctx context.Context, service *deployment.Service, targetID string,
) (CommandOutcome, error) {
	if service == nil {
		return CommandOutcome{}, errors.New("deployment service is required")
	}
	return service.Launch(ctx, targetID)
}
