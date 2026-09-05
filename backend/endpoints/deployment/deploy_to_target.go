/*
Endpoint: DeployToTarget
EndpointID: deploy_to_target
Purpose: Prepares the current session state and safely replaces the save of one target, optionally starting the game afterwards.
How it works: The runtime handler runs the shared save preparation phase of SaveEngine into a temporary file, then hands that file to the deployment service, which takes the mandatory backup of an existing target save, transfers, verifies, atomically replaces and verifies again. The local source file is never written and the session is never marked clean.
Supported resource types: —.
Input variables: targetID, saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk, launchAfter, confirmations.
GameCatalog variables read: none.
Save variables read: the in-memory state of the session registered under saveSessionID, serialized and validated exactly as a Save would.
Implementation status: implemented
*/
package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// DeployToTargetEndpointID is the stable backend identifier of DeployToTarget.
const DeployToTargetEndpointID = "deploy_to_target"

// DeployToTargetDefinition describes the public mutation contract.
var DeployToTargetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeployToTarget",
	ID:                         DeployToTargetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "saveSessionID", "expectedRevision", "validationToken", "launchAfter"},
	Description:                "Prepares the current session state and safely replaces the save of one target, optionally starting the game afterwards.",
})

// OperationResult is the typed result of every long deployment operation.
type OperationResult = deployment.OperationResult

// DeployRequest carries one deployment together with the explicit decisions the
// user already made for it.
//
// The three confirmation flags are the answers to the three blocked outcomes the
// service can report. They are never defaulted to true here: a confirmation the
// user did not give must not be invented by a backend default.
type DeployRequest struct {
	OperationID      string `json:"operationID"`
	TargetID         string `json:"targetID"`
	SaveSessionID    string `json:"saveSessionID"`
	ExpectedRevision string `json:"expectedRevision"`
	ValidationToken  string `json:"validationToken"`
	ConfirmWarnings  bool   `json:"confirmWarnings,omitempty"`
	ConfirmBanRisk   bool   `json:"confirmBanRisk,omitempty"`
	LaunchAfter      bool   `json:"launchAfter,omitempty"`

	ContinueWithUnknownGameStatus bool `json:"continueWithUnknownGameStatus,omitempty"`
	ConfirmRemoteBackup           bool `json:"confirmRemoteBackup,omitempty"`
	ConfirmStopGame               bool `json:"confirmStopGame,omitempty"`
}

// DeployToTarget performs Upload and Deploy & Launch.
//
// The prepared file is written into a private temporary directory and removed
// again whatever the outcome, so a deployment never leaves a copy of a save
// lying around and never writes the user's own local file.
func DeployToTarget(
	ctx context.Context,
	service *deployment.Service,
	engine *saveengine.Engine,
	request DeployRequest,
) (OperationResult, error) {
	if service == nil {
		return OperationResult{}, errors.New("deployment service is required")
	}
	if engine == nil {
		return OperationResult{}, errors.New("saveengine.Engine is required")
	}
	staging, err := os.MkdirTemp("", "saveforge-deploy-")
	if err != nil {
		return OperationResult{}, errors.New("cannot prepare the deployment staging directory")
	}
	defer os.RemoveAll(staging)

	prepared := filepath.Join(staging, "prepared.sl2")
	if _, err := engine.ExportForDeployment(
		request.SaveSessionID,
		request.ExpectedRevision,
		request.ValidationToken,
		request.ConfirmWarnings,
		request.ConfirmBanRisk,
		prepared,
	); err != nil {
		return OperationResult{}, err
	}

	return service.Upload(ctx, deployment.OperationRequest{
		OperationID:                   request.OperationID,
		TargetID:                      request.TargetID,
		PreparedPath:                  prepared,
		ContinueWithUnknownGameStatus: request.ContinueWithUnknownGameStatus,
		ConfirmRemoteBackup:           request.ConfirmRemoteBackup,
		ConfirmStopGame:               request.ConfirmStopGame,
	}, request.LaunchAfter)
}
