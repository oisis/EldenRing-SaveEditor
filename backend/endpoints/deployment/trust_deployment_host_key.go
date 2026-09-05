/*
Endpoint: TrustDeploymentHostKey
EndpointID: trust_deployment_host_key
Purpose: Records the SSH host key fingerprint the user approved for one target.
How it works: The runtime handler stores the fingerprint against the target's address, and only when a handshake with that exact address actually presented it. A fingerprint no connection observed is refused, so an approval can never be given for an invented value or for a different host. Nothing in the connection path ever records an approval on its own.
Supported resource types: —.
Input variables: targetID, fingerprint.
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

// TrustDeploymentHostKeyEndpointID is the stable backend identifier of TrustDeploymentHostKey.
const TrustDeploymentHostKeyEndpointID = "trust_deployment_host_key"

// TrustDeploymentHostKeyDefinition describes the public mutation contract.
var TrustDeploymentHostKeyDefinition = contract.MustDefine(contract.Definition{
	Name:                       "TrustDeploymentHostKey",
	ID:                         TrustDeploymentHostKeyEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "fingerprint"},
	Description:                "Records the SSH host key fingerprint the user approved for one target.",
})

// TrustDeploymentHostKey records an approved fingerprint.
func TrustDeploymentHostKey(
	store *deployment.Store, targetID string, fingerprint string,
) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	target, err := store.GetTarget(targetID)
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	if target.Kind != deployment.KindSSH {
		return GetDeploymentTargetsResult{}, errors.New("only an SSH target has a host key")
	}
	if err := store.TrustHostKey(target.Address(), fingerprint); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}
