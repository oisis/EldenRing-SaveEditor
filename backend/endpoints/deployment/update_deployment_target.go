/*
Endpoint: UpdateDeploymentTarget
EndpointID: update_deployment_target
Purpose: Replaces the configuration of one deployment target.
How it works: The runtime handler validates the complete target and persists it. Moving an SSH target to another address drops the host key approved for the previous one.
Supported resource types: —.
Input variables: targetID, name, kind, savePath, startCommand, stopCommand, statusCommand, host, port, user, keyPath.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented
*/
package deployment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// UpdateDeploymentTargetEndpointID is the stable backend identifier of UpdateDeploymentTarget.
const UpdateDeploymentTargetEndpointID = "update_deployment_target"

// UpdateDeploymentTargetDefinition describes the public mutation contract.
var UpdateDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "UpdateDeploymentTarget",
	ID:                         UpdateDeploymentTargetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "name", "kind", "savePath", "startCommand", "stopCommand", "statusCommand", "host", "port", "user", "keyPath"},
	Description:                "Replaces the configuration of one deployment target.",
})

// UpdateDeploymentTarget replaces one target and reports the whole library.
func UpdateDeploymentTarget(
	store *deployment.Store, input TargetInput,
) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	if input.TargetID == "" {
		return GetDeploymentTargetsResult{}, errors.New("a deployment target update needs the target identifier")
	}
	target, err := targetFromInput(input)
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	if _, err := store.UpdateTarget(target); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}
