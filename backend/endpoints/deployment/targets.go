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

Endpoint: CreateDeploymentTarget
EndpointID: create_deployment_target
Purpose: Stores a new deployment target.
How it works: The runtime handler validates the complete target, assigns the identifier itself and persists the configuration atomically.
Supported resource types: —.
Input variables: name, kind, savePath, startCommand, stopCommand, statusCommand, host, port, user, keyPath.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: UpdateDeploymentTarget
EndpointID: update_deployment_target
Purpose: Replaces the configuration of one deployment target.
How it works: The runtime handler validates the complete target and persists it. Moving an SSH target to another address drops the host key approved for the previous one.
Supported resource types: —.
Input variables: targetID, name, kind, savePath, startCommand, stopCommand, statusCommand, host, port, user, keyPath.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: DeleteDeploymentTarget
EndpointID: delete_deployment_target
Purpose: Removes one deployment target from the configuration.
How it works: The runtime handler drops the target, its approved host key and its backup metadata. No file on the target itself is touched.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: TestDeploymentTarget
EndpointID: test_deployment_target
Purpose: Verifies that one target is reachable and its save location usable.
How it works: The runtime handler asks the target's driver to prove the save directory exists and is writable, then reports the confirmed game status and whether a save is already present. For an SSH target this is also the handshake that observes the host key: an unapproved or changed key refuses the connection and is reported as a pending or changed host key together with the fingerprint the host actually presented. It writes no save and leaves the target unchanged.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented

Endpoint: TrustDeploymentHostKey
EndpointID: trust_deployment_host_key
Purpose: Records the SSH host key fingerprint the user approved for one target.
How it works: The runtime handler stores the fingerprint against the target's address, and only when a handshake with that exact address actually presented it. A fingerprint no connection observed is refused, so an approval can never be given for an invented value or for a different host. Nothing in the connection path ever records an approval on its own.
Supported resource types: —.
Input variables: targetID, fingerprint.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented

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
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// The stable backend identifiers of the target configuration endpoints.
const (
	GetDeploymentTargetsEndpointID    = "get_deployment_targets"
	CreateDeploymentTargetEndpointID  = "create_deployment_target"
	UpdateDeploymentTargetEndpointID  = "update_deployment_target"
	DeleteDeploymentTargetEndpointID  = "delete_deployment_target"
	TestDeploymentTargetEndpointID    = "test_deployment_target"
	TrustDeploymentHostKeyEndpointID  = "trust_deployment_host_key"
	ForgetDeploymentHostKeyEndpointID = "forget_deployment_host_key"
)

var (
	GetDeploymentTargetsDefinition = contract.MustDefine(contract.Definition{
		Name:                   "GetDeploymentTargets",
		ID:                     GetDeploymentTargetsEndpointID,
		Kind:                   contract.Getter,
		SupportedResourceTypes: "—",
		Description:            "Returns the configured deployment targets together with the trust state of each SSH host key.",
	})
	CreateDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
		Name:                       "CreateDeploymentTarget",
		ID:                         CreateDeploymentTargetEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"name", "kind", "savePath", "startCommand", "stopCommand", "statusCommand", "host", "port", "user", "keyPath"},
		Description:                "Stores a new deployment target.",
	})
	UpdateDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
		Name:                       "UpdateDeploymentTarget",
		ID:                         UpdateDeploymentTargetEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "name", "kind", "savePath", "startCommand", "stopCommand", "statusCommand", "host", "port", "user", "keyPath"},
		Description:                "Replaces the configuration of one deployment target.",
	})
	DeleteDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
		Name:                       "DeleteDeploymentTarget",
		ID:                         DeleteDeploymentTargetEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Removes one deployment target from the configuration.",
	})
	TestDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
		Name:                       "TestDeploymentTarget",
		ID:                         TestDeploymentTargetEndpointID,
		Kind:                       contract.Getter,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Verifies that one target is reachable and its save location usable.",
	})
	TrustDeploymentHostKeyDefinition = contract.MustDefine(contract.Definition{
		Name:                       "TrustDeploymentHostKey",
		ID:                         TrustDeploymentHostKeyEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "fingerprint"},
		Description:                "Records the SSH host key fingerprint the user approved for one target.",
	})
	ForgetDeploymentHostKeyDefinition = contract.MustDefine(contract.Definition{
		Name:                       "ForgetDeploymentHostKey",
		ID:                         ForgetDeploymentHostKeyEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Drops the SSH host key fingerprint approved for one target.",
	})
)

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

// GetDeploymentTargetsResult is the typed result of GetDeploymentTargets.
type GetDeploymentTargetsResult struct {
	Targets []TargetEntry `json:"targets"`
	// AvailableKinds is the closed target vocabulary, in interface order.
	AvailableKinds []string `json:"availableKinds"`
}

// TargetInput is the complete configuration of one target as a caller states
// it. Create ignores TargetID; Update requires it.
type TargetInput struct {
	TargetID     string `json:"targetID,omitempty"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	SavePath     string `json:"savePath"`
	StartCommand string `json:"startCommand,omitempty"`
	StopCommand  string `json:"stopCommand,omitempty"`
	// StatusCommand states whether the game is running. Its contract is the exit
	// code: 0 running, 1 stopped, anything else unknown.
	StatusCommand string `json:"statusCommand,omitempty"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	User          string `json:"user,omitempty"`
	KeyPath       string `json:"keyPath,omitempty"`
}

func (input TargetInput) target() (deployment.Target, error) {
	kind, err := deployment.ParseKind(input.Kind)
	if err != nil {
		return deployment.Target{}, err
	}
	return deployment.Target{
		ID:            input.TargetID,
		Name:          input.Name,
		Kind:          kind,
		SavePath:      input.SavePath,
		StartCommand:  input.StartCommand,
		StopCommand:   input.StopCommand,
		StatusCommand: input.StatusCommand,
		Host:          input.Host,
		Port:          input.Port,
		User:          input.User,
		KeyPath:       input.KeyPath,
	}, nil
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

// CreateDeploymentTarget stores a new target and reports the whole library.
func CreateDeploymentTarget(
	store *deployment.Store, input TargetInput,
) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	target, err := input.target()
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	if _, err := store.CreateTarget(target); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}

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
	target, err := input.target()
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	if _, err := store.UpdateTarget(target); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}

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
