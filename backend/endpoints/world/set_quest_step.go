/*
Endpoint: SetQuestStep
EndpointID: set_quest_step
Purpose: Sets an NPC questline to an explicitly specified supported step under expectedRevision control.
How it works: The handler validates the requested quest and step through GameCatalog, prepares the canonical event flag plan, and delegates one atomic event flag mutation to SaveEngine under expectedRevision control.
Supported resource types: QuestDocument.
Input variables: saveSessionID, characterID, questKind, questKey, stepKind, stepKey, expectedRevision.
GameCatalog variables read: the declared quest resource, its supported steps and their canonical event flag plans.
Save variables processed: the event flag bits of the requested slot's bitfield; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetQuestStepEndpointID is the stable backend identifier of SetQuestStep.
const SetQuestStepEndpointID = "set_quest_step"

// SetQuestStepDefinition describes the public mutation contract.
var SetQuestStepDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetQuestStep",
	ID:                         SetQuestStepEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "QuestDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "questKind", "questKey", "stepKind", "stepKey", "expectedRevision"},
	Description:                "Sets an NPC questline to an explicitly specified supported step under expectedRevision control.",
})

// SetQuestStepResult reports the committed state in public catalog terms.
// SaveEngine supplies the session state; this endpoint adds the catalog identity
// it resolved without exposing private event flags or offsets.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetQuestStepResult struct {
	saveengine.MutationReceipt
	CharacterID int                 `json:"characterID"`
	QuestKind   schema.ResourceKind `json:"questKind"`
	QuestKey    string              `json:"questKey"`
	StepKind    string              `json:"stepKind"`
	StepKey     string              `json:"stepKey"`
}

const questStepKind = "quest_step"

// SetQuestStep applies the curated event flag plan of one supported quest step
// to a character slot in an existing save session under expectedRevision control.
func SetQuestStep(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	questKind string,
	questKey string,
	stepKind string,
	stepKey string,
	expectedRevision string,
) (SetQuestStepResult, error) {
	if engine == nil {
		return SetQuestStepResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetQuestStepResult{}, errors.New("game catalog is not available")
	}
	if questKind != string(schema.ResourceKindQuest) {
		return SetQuestStepResult{}, fmt.Errorf(
			"resource kind %q is not %q", questKind, schema.ResourceKindQuest)
	}
	if stepKind != questStepKind {
		return SetQuestStepResult{}, fmt.Errorf(
			"step kind %q is not %q", stepKind, questStepKind)
	}

	resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKindQuest, questKey)
	if err != nil {
		return SetQuestStepResult{}, fmt.Errorf("unknown resource key %q in kind %q", questKey, questKind)
	}
	if resource.Quest == nil {
		return SetQuestStepResult{}, fmt.Errorf("resource %q carries no quest document", questKey)
	}

	var matchedStep *schema.QuestStepDocument
	for i := range resource.Quest.Steps {
		if resource.Quest.Steps[i].Key == stepKey {
			matchedStep = &resource.Quest.Steps[i]
			break
		}
	}
	if matchedStep == nil {
		return SetQuestStepResult{}, fmt.Errorf(
			"unknown step key %q in quest %q", stepKey, questKey)
	}

	plan := make([]saveengine.QuestFlagTarget, len(matchedStep.Flags))
	for i, flag := range matchedStep.Flags {
		plan[i] = saveengine.QuestFlagTarget{
			ID:    flag.ID,
			Value: flag.Value,
		}
	}

	mutation, err := engine.SetQuestStep(
		saveSessionID,
		characterID,
		plan,
		expectedRevision,
	)
	if err != nil {
		return SetQuestStepResult{}, err
	}

	return SetQuestStepResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		QuestKind:       schema.ResourceKindQuest,
		QuestKey:        questKey,
		StepKind:        questStepKind,
		StepKey:         stepKey,
	}, nil
}
