/*
Endpoint: GetDeploymentGameStatus
EndpointID: get_deployment_game_status
Purpose: Reports the confirmed game state of one deployment target.
How it works: The runtime handler runs the status command the target explicitly configures and maps its exit code: 0 means running, 1 means stopped, and every other outcome - no configured command, another exit code, a timeout, a transport fault or a command that could not be started - is unknown. It never guesses a state from a process name, from the start command or from the save's modification time.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented
*/
package deployment

import (
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetDeploymentGameStatusEndpointID is the stable backend identifier of GetDeploymentGameStatus.
const GetDeploymentGameStatusEndpointID = "get_deployment_game_status"

// GetDeploymentGameStatusDefinition describes the public getter contract.
var GetDeploymentGameStatusDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetDeploymentGameStatus",
	ID:                         GetDeploymentGameStatusEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Reports the confirmed game state of one deployment target.",
})

// GetDeploymentGameStatusResult is the typed result of GetDeploymentGameStatus.
type GetDeploymentGameStatusResult struct {
	TargetID   string `json:"targetID"`
	GameStatus string `json:"gameStatus"`
}

// GetDeploymentGameStatus reports the confirmed game state of one target.
func GetDeploymentGameStatus(
	ctx context.Context, service *deployment.Service, targetID string,
) (GetDeploymentGameStatusResult, error) {
	if service == nil {
		return GetDeploymentGameStatusResult{}, errors.New("deployment service is required")
	}
	status, err := service.GameStatusOf(ctx, targetID)
	if err != nil {
		return GetDeploymentGameStatusResult{}, err
	}
	return GetDeploymentGameStatusResult{TargetID: targetID, GameStatus: string(status)}, nil
}
