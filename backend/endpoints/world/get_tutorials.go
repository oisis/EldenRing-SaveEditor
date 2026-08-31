/*
Endpoint: GetTutorials
EndpointID: get_tutorials
Purpose: Returns user-facing tutorials and whether they have been unlocked.
How it works: The handler validates every TutorialDocument, resolves their unique TutorialParam row IDs, reads TutorialData once through SaveEngine and applies the exact availability filter without decoding save bytes itself.
Supported resource types: TutorialDocument.
Input variables: saveSessionID, characterID, availabilityFilter.
GameCatalog variables read: resource kind and key plus tutorial title and TutorialParam row ID.
Save variables read: the character activity flag and TutorialData membership; the getter writes nothing.
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

// GetTutorialsEndpointID is the stable backend identifier of GetTutorials.
const GetTutorialsEndpointID = "get_tutorials"

// Tutorial availability filters are matched exactly. The empty string means all
// tutorials and is deliberately not given a second spelling.
const (
	TutorialAvailabilityUnlocked = "unlocked"
	TutorialAvailabilityLocked   = "locked"
)

// GetTutorialsDefinition describes the public getter contract.
var GetTutorialsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetTutorials",
	ID:                         GetTutorialsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "TutorialDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "availabilityFilter"},
	Description:                "Returns user-facing tutorials and whether they have been unlocked.",
})

// TutorialEntry is one catalog-declared tutorial and its current membership in
// TutorialData. The row ID is represented once by the decimal resource key,
// without a second numeric field.
type TutorialEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Title    string              `json:"title"`
	Unlocked bool                `json:"unlocked"`
}

// GetTutorialsResult is the deterministic result for one character slot.
type GetTutorialsResult struct {
	SaveSessionID string          `json:"saveSessionID"`
	SaveRevision  string          `json:"saveRevision"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Tutorials     []TutorialEntry `json:"tutorials"`
}

// GetTutorials joins the catalog declarations with TutorialData membership.
// Catalog conflicts and an invalid filter fail before the save is read. An
// inactive slot reports every tutorial locked before filtering, without reading
// residual slot contents.
func GetTutorials(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetTutorialsResult, error) {
	if engine == nil {
		return GetTutorialsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetTutorialsResult{}, errors.New("game catalog is not available")
	}
	switch availabilityFilter {
	case "", TutorialAvailabilityUnlocked, TutorialAvailabilityLocked:
	default:
		return GetTutorialsResult{}, fmt.Errorf(
			"availabilityFilter must be %q, %q or empty; got %q",
			TutorialAvailabilityUnlocked, TutorialAvailabilityLocked, availabilityFilter)
	}

	declared, err := catalogTutorials(gameCatalog)
	if err != nil {
		return GetTutorialsResult{}, err
	}
	state, err := engine.GetTutorialIDs(saveSessionID, characterID)
	if err != nil {
		return GetTutorialsResult{}, err
	}
	unlockedIDs := make(map[uint32]struct{}, len(state.IDs))
	for _, tutorialID := range state.IDs {
		unlockedIDs[tutorialID] = struct{}{}
	}

	result := GetTutorialsResult{
		SaveSessionID: state.SaveSessionID,
		SaveRevision:  state.SaveRevision,
		CharacterID:   state.CharacterID,
		Active:        state.Active,
		Tutorials:     make([]TutorialEntry, 0, len(declared)),
	}
	for _, tutorial := range declared {
		entry := tutorial.entry
		_, entry.Unlocked = unlockedIDs[tutorial.tutorialID]
		if availabilityFilter == TutorialAvailabilityUnlocked && !entry.Unlocked {
			continue
		}
		if availabilityFilter == TutorialAvailabilityLocked && entry.Unlocked {
			continue
		}
		result.Tutorials = append(result.Tutorials, entry)
	}
	return result, nil
}

type declaredTutorial struct {
	entry      TutorialEntry
	tutorialID uint32
}

// catalogTutorials returns tutorials ordered by title, then key. Catalog
// validation requires the key to equal the TutorialParam row ID, so the normal
// unique resource identity also proves unique physical IDs.
func catalogTutorials(gameCatalog *gamecatalog.Catalog) ([]declaredTutorial, error) {
	declared := make([]declaredTutorial, 0)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindTutorial {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("tutorial %q: %w", summary.Key, err)
		}
		if resource.Tutorial == nil {
			return nil, fmt.Errorf("tutorial %q carries no tutorial document", summary.Key)
		}
		tutorialID := resource.Tutorial.TutorialID.Value
		declared = append(declared, declaredTutorial{
			entry: TutorialEntry{
				Kind:  resource.Kind,
				Key:   resource.Key,
				Title: resource.Tutorial.Title.Value,
			},
			tutorialID: tutorialID,
		})
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].entry.Title != declared[j].entry.Title {
			return declared[i].entry.Title < declared[j].entry.Title
		}
		return declared[i].entry.Key < declared[j].entry.Key
	})
	return declared, nil
}
