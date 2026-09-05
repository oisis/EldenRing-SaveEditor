/*
Endpoint: TestDeploymentTarget
EndpointID: test_deployment_target
Purpose: Verifies that one target is reachable and its save location usable.
How it works: The runtime handler asks the target's driver to prove the save directory exists and is writable, then reports the confirmed game status and whether a save is already present. For an SSH target this is also the handshake that observes the host key: an unapproved or changed key refuses the connection and is reported as a pending or changed host key together with the fingerprint the host actually presented. It writes no save and leaves the target unchanged.
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

// TestDeploymentTargetEndpointID is the stable backend identifier of TestDeploymentTarget.
const TestDeploymentTargetEndpointID = "test_deployment_target"

// TestDeploymentTargetDefinition describes the public getter contract.
var TestDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "TestDeploymentTarget",
	ID:                         TestDeploymentTargetEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Verifies that one target is reachable and its save location usable.",
})

// TestDeploymentTargetResult is the typed result of TestDeploymentTarget.
type TestDeploymentTargetResult = deployment.TestTargetResult

// TestDeploymentTarget verifies one target.
func TestDeploymentTarget(
	ctx context.Context, service *deployment.Service, targetID string,
) (TestDeploymentTargetResult, error) {
	if service == nil {
		return TestDeploymentTargetResult{}, errors.New("deployment service is required")
	}
	return service.TestTarget(ctx, targetID)
}
