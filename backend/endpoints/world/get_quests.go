/*
Endpoint: GetQuests
EndpointID: get_quests
Purpose: Returns the curated questlines, their supported steps and whether each step's canonical event flag plan currently matches the save.
How it works: The handler resolves the requested quest resources through GameCatalog, collects every distinct event flag their step plans declare and resolves all of them in one bulk SaveEngine read. It decodes no flag itself and reports one independent match per step; it declares no current step and no transition graph.
Supported resource types: QuestDocument.
Input variables: saveSessionID, characterID, questKind, questKey.
GameCatalog variables read: resource kind and key plus the quest name and, per supported step, its key, description, location and canonical event flag plan.
Save variables read: the character activity flag and the requested event flag bits; the getter writes nothing.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetQuestsEndpointID is the stable backend identifier of GetQuests.
const GetQuestsEndpointID = "get_quests"

// GetQuestsDefinition describes the public getter contract. The old contract
// promised "allowed transitions", which no source declares: the curated quest
// data is a flat list of independently addressable steps with a canonical event
// flag plan each, and neither SaveForge 1.5.8 nor 1.6.8 ever owned a transition
// graph or a precondition rule. Deriving one from step order or index adjacency
// would invent a contract, so this getter reports match state only. The declared
// variables gain saveSessionID, because the state belongs to a session snapshot.
var GetQuestsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetQuests",
	ID:                         GetQuestsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "QuestDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "questKind", "questKey"},
	Description:                "Returns the curated questlines and whether each supported step's canonical event flag plan currently matches the save.",
})

// QuestStepEntry is one supported step of a questline. Matched is true when
// every event flag of the step's canonical plan currently holds its declared
// target value. It is a statement about the save state alone: several steps of
// one questline may match at the same time, and none of them is named the
// current step, because no confirmed source declares such a rule. The plan
// itself — the event flag IDs and their targets — stays private.
type QuestStepEntry struct {
	StepKind    string `json:"stepKind"`
	StepKey     string `json:"stepKey"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Matched     bool   `json:"matched"`
}

// QuestEntry is one catalog-declared questline with its supported steps in the
// curated catalog order.
type QuestEntry struct {
	Kind  schema.ResourceKind `json:"kind"`
	Key   string              `json:"key"`
	Name  string              `json:"name"`
	Steps []QuestStepEntry    `json:"steps"`
}

// GetQuestsResult is the deterministic result for one character slot.
type GetQuestsResult struct {
	SaveSessionID string       `json:"saveSessionID"`
	SaveRevision  string       `json:"saveRevision"`
	CharacterID   int          `json:"characterID"`
	Active        bool         `json:"active"`
	Quests        []QuestEntry `json:"quests"`
}

// GetQuests joins the curated questlines with the save-side event flags. An
// empty questKey returns every declared questline; a non-empty one returns
// exactly that questline or fails. questKind is always matched exactly, so an
// unrelated resource kind can never be answered with a quest state.
//
// The whole selection is validated and its distinct flags are collected before
// one save byte is touched, so all steps are answered from a single bulk read.
// An inactive or residual slot reports active false and every step unmatched
// without its slot data being read: an all-cleared plan must not be reported as
// matched from a bitfield that was never looked at.
func GetQuests(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	questKind string,
	questKey string,
) (GetQuestsResult, error) {
	if engine == nil {
		return GetQuestsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetQuestsResult{}, errors.New("game catalog is not available")
	}
	if questKind != string(schema.ResourceKindQuest) {
		return GetQuestsResult{}, fmt.Errorf(
			"resource kind %q is not %q", questKind, schema.ResourceKindQuest)
	}

	declared, err := catalogQuests(gameCatalog, questKey)
	if err != nil {
		return GetQuestsResult{}, err
	}
	requested := make(map[uint32]struct{})
	for _, quest := range declared {
		for _, step := range quest.Quest.Steps {
			for _, flag := range step.Flags {
				requested[flag.ID] = struct{}{}
			}
		}
	}
	eventFlagIDs := make([]uint32, 0, len(requested))
	for id := range requested {
		eventFlagIDs = append(eventFlagIDs, id)
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetQuestsResult{}, err
	}

	result := GetQuestsResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		Quests:        make([]QuestEntry, 0, len(declared)),
	}
	for _, quest := range declared {
		entry := QuestEntry{
			Kind:  quest.Kind,
			Key:   quest.Key,
			Name:  quest.Quest.Name.Value,
			Steps: make([]QuestStepEntry, 0, len(quest.Quest.Steps)),
		}
		for _, step := range quest.Quest.Steps {
			entry.Steps = append(entry.Steps, QuestStepEntry{
				StepKind:    questStepKind,
				StepKey:     step.Key,
				Description: step.Description.Value,
				Location:    step.Location.Value,
				Matched:     flags.Active && planMatches(step.Flags, flags.Flags),
			})
		}
		result.Quests = append(result.Quests, entry)
	}
	return result, nil
}

// planMatches reports whether every flag of one canonical step plan currently
// holds its declared target value. The catalog rejects an empty plan, so a step
// can never match vacuously.
func planMatches(plan []schema.QuestFlag, current map[uint32]bool) bool {
	for _, flag := range plan {
		if current[flag.ID] != flag.Value {
			return false
		}
	}
	return true
}

// catalogQuests returns the requested questlines ordered by name, then key. An
// empty questKey selects all of them. It fails closed on a resource whose quest
// document is missing.
func catalogQuests(gameCatalog *gamecatalog.Catalog, questKey string) ([]schema.Resource, error) {
	if questKey != "" {
		resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKindQuest, questKey)
		if err != nil {
			return nil, fmt.Errorf(
				"unknown resource key %q in kind %q", questKey, schema.ResourceKindQuest)
		}
		if resource.Quest == nil {
			return nil, fmt.Errorf("resource %q carries no quest document", questKey)
		}
		return []schema.Resource{resource}, nil
	}

	declared := make([]schema.Resource, 0)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindQuest {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("quest %q: %w", summary.Key, err)
		}
		if resource.Quest == nil {
			return nil, fmt.Errorf("quest %q carries no quest document", summary.Key)
		}
		declared = append(declared, resource)
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].Quest.Name.Value != declared[j].Quest.Name.Value {
			return declared[i].Quest.Name.Value < declared[j].Quest.Name.Value
		}
		return declared[i].Key < declared[j].Key
	})
	return declared, nil
}
