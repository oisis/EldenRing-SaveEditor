/*
Endpoint: ForgetDeploymentHostKey
EndpointID: forget_deployment_host_key
Purpose: Drops the SSH host key fingerprint approved for one target.
How it works: The runtime handler removes the stored fingerprint, so the next connection asks for an explicit approval again.
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

// ForgetDeploymentHostKeyEndpointID is the stable backend identifier of ForgetDeploymentHostKey.
const ForgetDeploymentHostKeyEndpointID = "forget_deployment_host_key"

// ForgetDeploymentHostKeyDefinition describes the public mutation contract.
var ForgetDeploymentHostKeyDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ForgetDeploymentHostKey",
	ID:                         ForgetDeploymentHostKeyEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Drops the SSH host key fingerprint approved for one target.",
})

// ForgetDeploymentHostKey drops an approved fingerprint.
func ForgetDeploymentHostKey(
	store *deployment.Store, targetID string,
) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	target, err := store.GetTarget(targetID)
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	if err := store.ForgetHostKey(target.Address()); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}
