/*
Endpoint: SetBossDefeated
EndpointID: set_boss_defeated
Purpose: Sets a boss's defeated state only under that resource's confirmed contract.
How it works: The handler validates the complete curated boss catalog, resolves the requested resource by its exact kind and key, and delegates one atomic event flag mutation to SaveEngine under expectedRevision control.
Supported resource types: BossDocument.
Input variables: saveSessionID, characterID, bossKind, bossKey, defeated, expectedRevision.
GameCatalog variables read: resource kind and key plus the boss.defeatEventFlagID of the requested boss.
Save variables processed: the synchronized defeat event flag bit of the requested slot's bitfield; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetBossDefeatedEndpointID is the stable backend identifier of SetBossDefeated.
const SetBossDefeatedEndpointID = "set_boss_defeated"

// SetBossDefeatedDefinition describes the public mutation contract.
var SetBossDefeatedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetBossDefeated",
	ID:                         SetBossDefeatedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "BossDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "bossKind", "bossKey", "defeated", "expectedRevision"},
	Description:                "Sets a boss's defeated state only under that resource's confirmed contract.",
})

// SetBossDefeatedResult reports the committed state in public catalog terms.
// SaveEngine supplies the session state; this endpoint adds the catalog identity
// it resolved without exposing the internal defeat event flag.
type SetBossDefeatedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	BossKind      schema.ResourceKind `json:"bossKind"`
	BossKey       string              `json:"bossKey"`
	Defeated      bool                `json:"defeated"`
}

// SetBossDefeated sets or clears the synchronized defeat state of one catalog
// boss in a character slot of an existing save session. It changes that one
// event flag and nothing else, exactly like SaveForge 1.5.8 and 1.6.8: no
// reward, no Remembrance item and no arena or world flag is touched.
func SetBossDefeated(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	bossKind string,
	bossKey string,
	defeated bool,
	expectedRevision string,
) (SetBossDefeatedResult, error) {
	if engine == nil {
		return SetBossDefeatedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetBossDefeatedResult{}, errors.New("game catalog is not available")
	}
	if bossKind != string(schema.ResourceKindBoss) {
		return SetBossDefeatedResult{}, fmt.Errorf(
			"resource kind %q is not %q", bossKind, schema.ResourceKindBoss)
	}

	// The shared resolver validates the whole curated list — a missing document
	// or two bosses claiming one flag — before a save byte is touched. It covers
	// every boss resource, so a key it does not carry is unknown.
	declared, err := catalogBosses(gameCatalog)
	if err != nil {
		return SetBossDefeatedResult{}, err
	}
	var matched declaredBoss
	found := false
	for _, boss := range declared {
		if boss.entry.Key == bossKey {
			matched = boss
			found = true
			break
		}
	}
	if !found {
		return SetBossDefeatedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q", bossKey, bossKind)
	}

	mutation, err := engine.SetBossDefeated(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		defeated,
		expectedRevision,
	)
	if err != nil {
		return SetBossDefeatedResult{}, err
	}
	return SetBossDefeatedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		BossKind:      matched.entry.Kind,
		BossKey:       matched.entry.Key,
		Defeated:      mutation.Defeated,
	}, nil
}
