/*
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
)

// DownloadFromTargetEndpointID is the stable backend identifier of DownloadFromTarget.
const DownloadFromTargetEndpointID = "download_from_target"

// DownloadFromTargetDefinition describes the public mutation contract.
var DownloadFromTargetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DownloadFromTarget",
	ID:                         DownloadFromTargetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "closeGameFirst"},
	Description:                "Copies the save of one target into a local staging file, optionally stopping the game first.",
})

// DownloadRequest carries one download and its confirmations.
type DownloadRequest struct {
	OperationID    string `json:"operationID"`
	TargetID       string `json:"targetID"`
	CloseGameFirst bool   `json:"closeGameFirst,omitempty"`

	ContinueWithUnknownGameStatus bool `json:"continueWithUnknownGameStatus,omitempty"`
	ConfirmStopGame               bool `json:"confirmStopGame,omitempty"`
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
