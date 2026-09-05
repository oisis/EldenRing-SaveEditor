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

Endpoint: LaunchTargetGame
EndpointID: launch_target_game
Purpose: Runs the configured start command of one target as an explicit user action.
How it works: The runtime handler executes exactly the command line the user configured and returns its real outcome.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: CloseTargetGame
EndpointID: close_target_game
Purpose: Runs the configured stop command of one target as an explicit user action.
How it works: The runtime handler executes exactly the command line the user configured and returns its real outcome, including the case where no process was found.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: DeployToTarget
EndpointID: deploy_to_target
Purpose: Prepares the current session state and safely replaces the save of one target, optionally starting the game afterwards.
How it works: The runtime handler runs the shared save preparation phase of SaveEngine into a temporary file, then hands that file to the deployment service, which takes the mandatory backup of an existing target save, transfers, verifies, atomically replaces and verifies again. The local source file is never written and the session is never marked clean.
Supported resource types: —.
Input variables: targetID, saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk, launchAfter, confirmations.
GameCatalog variables read: none.
Save variables read: the in-memory state of the session registered under saveSessionID, serialized and validated exactly as a Save would.
Implementation status: implemented

Endpoint: DownloadFromTarget
EndpointID: download_from_target
Purpose: Copies the save of one target into a local staging file, optionally stopping the game first.
How it works: The runtime handler blocks while the game is confirmed to be running, waits for the target save to stop changing, copies it into a temporary staging file and reports that path. It replaces no session by itself.
Supported resource types: —.
Input variables: targetID, closeGameFirst, confirmations.
GameCatalog variables read: none.
Save variables read: none of the open session; only the target's own file.
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

// The stable backend identifiers of the operation endpoints.
const (
	GetDeploymentGameStatusEndpointID = "get_deployment_game_status"
	LaunchTargetGameEndpointID        = "launch_target_game"
	CloseTargetGameEndpointID         = "close_target_game"
	DeployToTargetEndpointID          = "deploy_to_target"
	DownloadFromTargetEndpointID      = "download_from_target"
)

var (
	GetDeploymentGameStatusDefinition = contract.MustDefine(contract.Definition{
		Name:                       "GetDeploymentGameStatus",
		ID:                         GetDeploymentGameStatusEndpointID,
		Kind:                       contract.Getter,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Reports the confirmed game state of one deployment target.",
	})
	LaunchTargetGameDefinition = contract.MustDefine(contract.Definition{
		Name:                       "LaunchTargetGame",
		ID:                         LaunchTargetGameEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Runs the configured start command of one target as an explicit user action.",
	})
	CloseTargetGameDefinition = contract.MustDefine(contract.Definition{
		Name:                       "CloseTargetGame",
		ID:                         CloseTargetGameEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Runs the configured stop command of one target as an explicit user action.",
	})
	DeployToTargetDefinition = contract.MustDefine(contract.Definition{
		Name:                       "DeployToTarget",
		ID:                         DeployToTargetEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "saveSessionID", "expectedRevision", "validationToken", "launchAfter"},
		Description:                "Prepares the current session state and safely replaces the save of one target, optionally starting the game afterwards.",
	})
	DownloadFromTargetDefinition = contract.MustDefine(contract.Definition{
		Name:                       "DownloadFromTarget",
		ID:                         DownloadFromTargetEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "closeGameFirst"},
		Description:                "Copies the save of one target into a local staging file, optionally stopping the game first.",
	})
)

// OperationResult is the typed result of every long deployment operation.
type OperationResult = deployment.OperationResult

// CommandOutcome is the typed result of Launch and Close Game.
type CommandOutcome = deployment.CommandOutcome

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

// DownloadRequest carries one download and its confirmations.
type DownloadRequest struct {
	OperationID    string `json:"operationID"`
	TargetID       string `json:"targetID"`
	CloseGameFirst bool   `json:"closeGameFirst,omitempty"`

	ContinueWithUnknownGameStatus bool `json:"continueWithUnknownGameStatus,omitempty"`
	ConfirmStopGame               bool `json:"confirmStopGame,omitempty"`
}

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

// LaunchTargetGame runs the configured start command.
func LaunchTargetGame(
	ctx context.Context, service *deployment.Service, targetID string,
) (CommandOutcome, error) {
	if service == nil {
		return CommandOutcome{}, errors.New("deployment service is required")
	}
	return service.Launch(ctx, targetID)
}

// CloseTargetGame runs the configured stop command.
func CloseTargetGame(
	ctx context.Context, service *deployment.Service, targetID string,
) (CommandOutcome, error) {
	if service == nil {
		return CommandOutcome{}, errors.New("deployment service is required")
	}
	return service.CloseGame(ctx, targetID)
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

// DownloadFromTarget performs Download and Close & Download.
//
// The staging file it produces is deliberately kept: the caller loads it as a
// temporary session, which is what keeps Save unavailable for it until an
// explicit Save As, exactly as section 9 of the deployment specification
// requires. A blocked or failed download removes it again.
func DownloadFromTarget(
	ctx context.Context, service *deployment.Service, request DownloadRequest,
) (OperationResult, error) {
	if service == nil {
		return OperationResult{}, errors.New("deployment service is required")
	}
	staging, err := os.MkdirTemp("", "saveforge-download-")
	if err != nil {
		return OperationResult{}, errors.New("cannot prepare the download staging directory")
	}
	result, err := service.Download(ctx, deployment.OperationRequest{
		OperationID:                   request.OperationID,
		TargetID:                      request.TargetID,
		StagingPath:                   filepath.Join(staging, "downloaded.sl2"),
		ContinueWithUnknownGameStatus: request.ContinueWithUnknownGameStatus,
		ConfirmStopGame:               request.ConfirmStopGame,
	}, request.CloseGameFirst)
	if err != nil || !result.Completed {
		os.RemoveAll(staging)
		if err != nil {
			return OperationResult{}, err
		}
		result.LocalPath = ""
	}
	return result, nil
}
