/*
Endpoint: SetTutorialUnlocked
EndpointID: set_tutorial_unlocked
Purpose: Sets the unlock state of a tutorial.
How it works: The runtime handler validates the request and the expected revision, resolves the tutorial resource by exact kind and key, passes the private TutorialParam row ID to SaveEngine under expectedRevision control, and returns the public kind and key receipt.
Supported resource types: TutorialDocument
Input variables: saveSessionID, characterID, tutorialKind, tutorialKey, unlocked, expectedRevision
GameCatalog variables read: resource kind and key plus the TutorialParam row ID of the requested tutorial.
Save variables processed: the TutorialData list of the requested character; SaveEngine validates expectedRevision, writes in place and finishes with full success or rollback.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetTutorialUnlockedEndpointID is the stable backend identifier of SetTutorialUnlocked.
const SetTutorialUnlockedEndpointID = "set_tutorial_unlocked"

// SetTutorialUnlockedDefinition describes the public mutation contract.
var SetTutorialUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                   "SetTutorialUnlocked",
	ID:                     SetTutorialUnlockedEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "TutorialDocument",
	SupportedResourceVariables: []string{
		"saveSessionID", "characterID", "tutorialKind", "tutorialKey", "unlocked", "expectedRevision"},
	Description: "Sets the unlock state of a tutorial.",
})

// SetTutorialUnlockedResult reports the committed state in public catalog terms.
type SetTutorialUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	TutorialKind  schema.ResourceKind `json:"tutorialKind"`
	TutorialKey   string              `json:"tutorialKey"`
	Unlocked      bool                `json:"unlocked"`
}

// SetTutorialUnlocked adds or removes one catalog-declared tutorial in the
// TutorialData list of a character slot of an existing save session.
//
// The private TutorialParam row ID is passed to SaveEngine and is never part of
// the result.
func SetTutorialUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	tutorialKind string,
	tutorialKey string,
	unlocked bool,
	expectedRevision string,
) (SetTutorialUnlockedResult, error) {
	if engine == nil {
		return SetTutorialUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetTutorialUnlockedResult{}, errors.New("game catalog is not available")
	}
	if tutorialKind != string(schema.ResourceKindTutorial) {
		return SetTutorialUnlockedResult{}, fmt.Errorf(
			"resource kind %q is not %q", tutorialKind, schema.ResourceKindTutorial)
	}

	resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKindTutorial, tutorialKey)
	if err != nil {
		return SetTutorialUnlockedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q: %w", tutorialKey, tutorialKind, err)
	}
	if resource.Tutorial == nil {
		return SetTutorialUnlockedResult{}, fmt.Errorf(
			"tutorial %q carries no tutorial document", tutorialKey)
	}
	if !resource.Tutorial.TutorialID.Known || resource.Tutorial.TutorialID.Value == 0 {
		return SetTutorialUnlockedResult{}, fmt.Errorf(
			"tutorial %q declares no confirmed non-zero tutorial ID", tutorialKey)
	}

	mutation, err := engine.SetTutorialUnlocked(
		saveSessionID,
		characterID,
		resource.Tutorial.TutorialID.Value,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetTutorialUnlockedResult{}, err
	}
	return SetTutorialUnlockedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		TutorialKind:  resource.Kind,
		TutorialKey:   resource.Key,
		Unlocked:      mutation.Unlocked,
	}, nil
}
