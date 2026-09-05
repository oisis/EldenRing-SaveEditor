/*
Endpoint: DeleteDeploymentTarget
EndpointID: delete_deployment_target
Purpose: Removes one deployment target from the configuration.
How it works: The runtime handler drops the target, its approved host key and its backup metadata. No file on the target itself is touched.
Supported resource types: —.
Input variables: targetID.
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

// DeleteDeploymentTargetEndpointID is the stable backend identifier of DeleteDeploymentTarget.
const DeleteDeploymentTargetEndpointID = "delete_deployment_target"

// DeleteDeploymentTargetDefinition describes the public mutation contract.
var DeleteDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteDeploymentTarget",
	ID:                         DeleteDeploymentTargetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Removes one deployment target from the configuration.",
})

// DeleteDeploymentTarget removes one target and reports the whole library.
func DeleteDeploymentTarget(
	store *deployment.Store, targetID string,
) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	if err := store.DeleteTarget(targetID); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}
