/*
Endpoint: CreateDeploymentTarget
EndpointID: create_deployment_target
Purpose: Stores a new deployment target.
How it works: The runtime handler validates the complete target, assigns the identifier itself and persists the configuration atomically.
Supported resource types: —.
Input variables: name, kind, savePath, startCommand, stopCommand, statusCommand, host, port, user, keyPath.
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

// CreateDeploymentTargetEndpointID is the stable backend identifier of CreateDeploymentTarget.
const CreateDeploymentTargetEndpointID = "create_deployment_target"

// CreateDeploymentTargetDefinition describes the public mutation contract.
var CreateDeploymentTargetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CreateDeploymentTarget",
	ID:                         CreateDeploymentTargetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"name", "kind", "savePath", "startCommand", "stopCommand", "statusCommand", "host", "port", "user", "keyPath"},
	Description:                "Stores a new deployment target.",
})

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

// targetFromInput parses the stated kind and builds the stored target. Create
// and Update share it so the two mutations can never disagree about what a
// stated configuration means.
func targetFromInput(input TargetInput) (deployment.Target, error) {
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

// CreateDeploymentTarget stores a new target and reports the whole library.
func CreateDeploymentTarget(
	store *deployment.Store, input TargetInput,
) (GetDeploymentTargetsResult, error) {
	if store == nil {
		return GetDeploymentTargetsResult{}, errors.New("deployment store is required")
	}
	target, err := targetFromInput(input)
	if err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	if _, err := store.CreateTarget(target); err != nil {
		return GetDeploymentTargetsResult{}, err
	}
	return GetDeploymentTargets(store)
}
