/*
Endpoint: GetDeploymentTargets
EndpointID: get_deployment_targets
Purpose: Returns the configured deployment targets together with the trust state of each SSH host key.
How it works: The runtime handler reads the deployment store and reports every target and, for an SSH target, whether a host key fingerprint has already been approved. It contacts no target and changes nothing.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none; a deployment target is host configuration, not save state.
Implementation status: implemented
*/
package deployment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetDeploymentTargetsEndpointID is the stable backend identifier of GetDeploymentTargets.
const GetDeploymentTargetsEndpointID = "get_deployment_targets"

// GetDeploymentTargetsDefinition describes the public getter contract.
var GetDeploymentTargetsDefinition = contract.MustDefine(contract.Definition{
	Name:                   "GetDeploymentTargets",
	ID:                     GetDeploymentTargetsEndpointID,
	Kind:                   contract.Getter,
	SupportedResourceTypes: "—",
	Description:            "Returns the configured deployment targets together with the trust state of each SSH host key.",
})

// TargetEntry is one configured target as the frontend sees it.
//
// The key path is reported so the configuration form can show what the user
// stated. The key's contents are never read: this application only ever holds a
// path, and the path is excluded from logs and from the diagnostic report.
type TargetEntry struct {
	deployment.Target
	// HostKeyTrusted reports whether an SSH host key fingerprint was approved
	// for this target's address. It is false for a local target.
	HostKeyTrusted bool `json:"hostKeyTrusted"`
	// HostKeyFingerprint is the approved fingerprint, empty when none is stored.
	HostKeyFingerprint string `json:"hostKeyFingerprint,omitempty"`
	// TransferSupported is false while a target kind has no safe transfer
	// implementation in this build. The interface disables the operations rather
	// than offering an action that must fail. Both supported kinds now have one,
	// so it is true for every stored target.
	TransferSupported bool `json:"transferSupported"`
	// UnsupportedReason explains a false TransferSupported in one safe sentence.
	UnsupportedReason string `json:"unsupportedReason,omitempty"`
}

// GetDeploymentTargetsResult is the typed result of GetDeploymentTargets. Every
// target mutation below reports the whole library through it, so a caller never
// has to merge a partial answer into its own copy.
type GetDeploymentTargetsResult struct {
	Targets []TargetEntry `json:"targets"`
	// AvailableKinds is the closed target vocabulary, in interface order.
	AvailableKinds []string `json:"availableKinds"`
}

// GetDeploymentTargets reports the configured targets.
func GetDeploymentTargets(store *deployment.Store) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	targets, err := store.ListTargets()
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	entries := make([]TargetEntry, 0, len(targets))
	for _, target := range targets {
		entry := TargetEntry{
			Target:            target,
			TransferSupported: deployment.TransferSupported(target.Kind),
		}
		if target.Kind == deployment.KindSSH {
			fingerprint, trusted, keyErr := store.TrustedHostKey(target.Address())
			if keyErr != nil {
				return GetDeploymentTargetsResult{}, keyErr
			}
			entry.HostKeyTrusted = trusted
			entry.HostKeyFingerprint = fingerprint
		}
		entries = append(entries, entry)
	}
	kinds := deployment.Kinds()
	available := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		available = append(available, string(kind))
	}
	return GetDeploymentTargetsResult{Targets: entries, AvailableKinds: available}, nil
}
