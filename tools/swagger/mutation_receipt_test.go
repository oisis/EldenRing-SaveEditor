package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/favorites"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/world"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// receiptMigratedResults are the mutation result schemas that carry the shared
// MutationReceipt today. Stage 3b.1 migrated the SaveSession and Network batch,
// stage 3b.2 added the Character, Appearance, Templates and Favorites batch,
// stage 3b.3a added the twelve Inventory mutations, stage 3b.3b added the seven
// Equipment mutations and stage 3b.3c added the fifteen World mutations.
// Diagnostics still returns its old shape, and this list must grow only together
// with its migration.
var receiptMigratedResults = []string{
	// Stage 3b.1: SaveSession and Network.
	"WriteSaveResult",
	"SetSaveAccountIDResult",
	"SetNetworkSettingsResult",
	"ApplyNetworkPresetResult",
	// Stage 3b.2: Character.
	"CloneCharacterResult",
	"DeleteCharacterResult",
	"SetCharacterAppearanceResult",
	"SetCharacterGenderResult",
	"SetCharacterNameResult",
	"SetCharacterRunesResult",
	"SetCharacterStartingClassResult",
	"SetCharacterStatsResult",
	"UndoCharacterChangesResult",
	// Stage 3b.2: Appearance, Templates and Favorites.
	"ApplyAppearancePresetResult",
	"ApplyBuildTemplateResult",
	"ApplyFavoritePresetResult",
	"DeleteFavoritePresetResult",
	"SetFavoritePresetResult",
	// Stage 3b.3a: Inventory.
	"AddItemToInventoryResult",
	"AddItemToStorageResult",
	"MoveOwnedItemToInventoryResult",
	"MoveOwnedItemToStorageResult",
	"RemoveOwnedItemResult",
	"SetOwnedItemQuantityResult",
	"SetInventoryOrderResult",
	"SetStorageOrderResult",
	"SetWeaponAshOfWarResult",
	"SetWeaponInfusionResult",
	"SetWeaponUpgradeLevelResult",
	"SetSpiritAshUpgradeLevelResult",
	// Stage 3b.3b: Equipment.
	"SetEquippedArmamentsResult",
	"SetEquippedArmorResult",
	"SetEquippedTalismansResult",
	"SetEquippedSpellsResult",
	"SetPhysickMixtureResult",
	"SetPouchItemsResult",
	"SetQuickItemsResult",
	// Stage 3b.3c: World.
	"LockAllSpectralSteedAttiresResult",
	"SetBellBearingUnlockedResult",
	"SetBossDefeatedResult",
	"SetColosseumUnlockedResult",
	"SetCookbookUnlockedResult",
	"SetFogOfWarRemovedResult",
	"SetGestureUnlockedResult",
	"SetGraceVisitedResult",
	"SetMapRegionRevealedResult",
	"SetQuestStepResult",
	"SetRegionUnlockedResult",
	"SetSpectralSteedAttireResult",
	"SetSummoningPoolActivatedResult",
	"SetTutorialUnlockedResult",
	"SetWhetbladeUnlockedResult",
	// Stage 3b.3d and the no-commit contract: the two endpoints with a success
	// that commits nothing describe their committed variant as its own schema, so
	// the complete receipt is required there and only there.
	"ApplyRepairsApplied",
	"SetCharacterActiveCommitted",
}

// receiptProperties are the five members every migrated result exposes flat.
var receiptProperties = []string{
	"operationID", "operationKind", "saveSessionID", "saveRevision", "changedScopes",
}

// The shared schema is the one source of truth for the receipt. Every migrated
// result must compose it instead of restating its five properties, and no
// migrated result may reintroduce saveSessionID or saveRevision of its own.
func TestOpenAPIMigratedMutationResultsRequireTheSharedReceipt(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Components struct {
			Schemas map[string]struct {
				Type       string                     `json:"type"`
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
				AllOf      []struct {
					Ref        string                     `json:"$ref"`
					Type       string                     `json:"type"`
					Properties map[string]json.RawMessage `json:"properties"`
					Required   []string                   `json:"required"`
				} `json:"allOf"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	schemas := document.Components.Schemas

	receipt, exists := schemas["MutationReceipt"]
	if !exists {
		t.Fatal("openapi.json describes no shared MutationReceipt schema")
	}
	for _, property := range receiptProperties {
		if _, declared := receipt.Properties[property]; !declared {
			t.Errorf("MutationReceipt declares no %q property", property)
		}
	}
	if !reflect.DeepEqual(receipt.Required, receiptProperties) {
		t.Errorf("MutationReceipt required = %v, want all five members %v",
			receipt.Required, receiptProperties)
	}

	const ref = "#/components/schemas/MutationReceipt"
	for _, name := range receiptMigratedResults {
		schema, exists := schemas[name]
		if !exists {
			t.Errorf("openapi.json describes no %s", name)
			continue
		}
		composed := false
		for _, part := range schema.AllOf {
			if part.Ref == ref {
				composed = true
			}
			for _, property := range receiptProperties {
				if _, restated := part.Properties[property]; restated {
					t.Errorf("%s restates receipt property %q instead of composing %s",
						name, property, ref)
				}
			}
		}
		if !composed {
			t.Errorf("%s does not compose %s, so its receipt is not required", name, ref)
		}
		for _, property := range receiptProperties {
			if _, restated := schema.Properties[property]; restated {
				t.Errorf("%s restates receipt property %q at its own level", name, property)
			}
		}
	}

	// A result schema that composes the receipt without being listed here is a
	// migration nobody recorded. The list is the reviewed set, not a lower bound.
	for name, schema := range schemas {
		composes := false
		for _, part := range schema.AllOf {
			if part.Ref == ref {
				composes = true
			}
		}
		if !composes {
			continue
		}
		listed := false
		for _, migrated := range receiptMigratedResults {
			if migrated == name {
				listed = true
			}
		}
		if !listed {
			t.Errorf("%s composes %s but is not part of the recorded migrated batch", name, ref)
		}
	}
}

// Every migrated result must report its own EndpointID as operationKind. Sharing
// a SaveEngine writer, as the two network endpoints and the three appearance
// entry points do, must not merge two kinds into one.
func TestMigratedMutationKindsAreTheirOwnEndpointIDs(t *testing.T) {
	kinds := map[string]string{
		"write_save":           savesession.WriteSaveEndpointID,
		"set_save_account_id":  savesession.SetSaveAccountIDEndpointID,
		"set_network_settings": network.SetNetworkSettingsEndpointID,
		"apply_network_preset": network.ApplyNetworkPresetEndpointID,

		"clone_character":              character.CloneCharacterEndpointID,
		"delete_character":             character.DeleteCharacterEndpointID,
		"set_character_appearance":     character.SetCharacterAppearanceEndpointID,
		"set_character_gender":         character.SetCharacterGenderEndpointID,
		"set_character_name":           character.SetCharacterNameEndpointID,
		"set_character_runes":          character.SetCharacterRunesEndpointID,
		"set_character_starting_class": character.SetCharacterStartingClassEndpointID,
		"set_character_stats":          character.SetCharacterStatsEndpointID,
		"undo_character_changes":       character.UndoCharacterChangesEndpointID,

		"apply_appearance_preset": appearance.ApplyAppearancePresetEndpointID,
		"apply_build_template":    templates.ApplyBuildTemplateEndpointID,
		"apply_favorite_preset":   favorites.ApplyFavoritePresetEndpointID,
		"delete_favorite_preset":  favorites.DeleteFavoritePresetEndpointID,
		"set_favorite_preset":     favorites.SetFavoritePresetEndpointID,

		"add_item_to_inventory":        inventory.AddItemToInventoryEndpointID,
		"add_item_to_storage":          inventory.AddItemToStorageEndpointID,
		"move_owned_item_to_inventory": inventory.MoveOwnedItemToInventoryEndpointID,
		"move_owned_item_to_storage":   inventory.MoveOwnedItemToStorageEndpointID,
		"remove_owned_item":            inventory.RemoveOwnedItemEndpointID,
		"set_owned_item_quantity":      inventory.SetOwnedItemQuantityEndpointID,
		"set_inventory_order":          inventory.SetInventoryOrderEndpointID,
		"set_storage_order":            inventory.SetStorageOrderEndpointID,
		// Every result keeps its own public EndpointID as operationKind. That
		// matters twice over for set_weapon_infusion and set_weapon_upgrade_level,
		// which are the two callers of the shared setOwnedWeaponGameID writer.
		"set_weapon_ash_of_war":        inventory.SetWeaponAshOfWarEndpointID,
		"set_weapon_infusion":          inventory.SetWeaponInfusionEndpointID,
		"set_weapon_upgrade_level":     inventory.SetWeaponUpgradeLevelEndpointID,
		"set_spirit_ash_upgrade_level": inventory.SetSpiritAshUpgradeLevelEndpointID,

		// The seven Equipment mutations each own a separate SaveEngine writer, so
		// none of them can inherit another endpoint's kind.
		"set_equipped_armaments": equipment.SetEquippedArmamentsEndpointID,
		"set_equipped_armor":     equipment.SetEquippedArmorEndpointID,
		"set_equipped_talismans": equipment.SetEquippedTalismansEndpointID,
		"set_equipped_spells":    equipment.SetEquippedSpellsEndpointID,
		"set_physick_mixture":    equipment.SetPhysickMixtureEndpointID,
		"set_pouch_items":        equipment.SetPouchItemsEndpointID,
		"set_quick_items":        equipment.SetQuickItemsEndpointID,

		// The fifteen World mutations. set_spectral_steed_attire and
		// lock_all_spectral_steed_attires share one SaveEngine result type, yet
		// each keeps its own operationKind and its own changedScopes.
		"lock_all_spectral_steed_attires": world.LockAllSpectralSteedAttiresEndpointID,
		"set_bell_bearing_unlocked":       world.SetBellBearingUnlockedEndpointID,
		"set_boss_defeated":               world.SetBossDefeatedEndpointID,
		"set_colosseum_unlocked":          world.SetColosseumUnlockedEndpointID,
		"set_cookbook_unlocked":           world.SetCookbookUnlockedEndpointID,
		"set_fog_of_war_removed":          world.SetFogOfWarRemovedEndpointID,
		"set_gesture_unlocked":            world.SetGestureUnlockedEndpointID,
		"set_grace_visited":               world.SetGraceVisitedEndpointID,
		"set_map_region_revealed":         world.SetMapRegionRevealedEndpointID,
		"set_quest_step":                  world.SetQuestStepEndpointID,
		"set_region_unlocked":             world.SetRegionUnlockedEndpointID,
		"set_spectral_steed_attire":       world.SetSpectralSteedAttireEndpointID,
		"set_summoning_pool_activated":    world.SetSummoningPoolActivatedEndpointID,
		"set_tutorial_unlocked":           world.SetTutorialUnlockedEndpointID,
		"set_whetblade_unlocked":          world.SetWhetbladeUnlockedEndpointID,
	}
	registered := make(map[string]bool)
	for _, kind := range saveengine.MutationKinds() {
		registered[kind] = true
	}
	for want, endpointID := range kinds {
		if endpointID != want {
			t.Errorf("EndpointID = %q, want %q", endpointID, want)
		}
		if !registered[endpointID] {
			t.Errorf("SaveEngine registers no operation kind %q", endpointID)
		}
	}
}

// assertRouteReceipt checks the complete receipt one transport-exposed mutation
// returned: a minted operationID, the endpoint's own kind, the session, the new
// revision and the exact canonical scopes of that kind.
func assertRouteReceipt(
	t *testing.T,
	receipt saveengine.MutationReceipt,
	saveSessionID string,
	operationKind string,
	saveRevision string,
) {
	t.Helper()

	if receipt.OperationID == "" {
		t.Errorf("receipt = %+v, want a minted operationID", receipt)
	}
	if receipt.OperationKind != operationKind {
		t.Errorf("operationKind = %q, want the EndpointID %q", receipt.OperationKind, operationKind)
	}
	if receipt.SaveSessionID != saveSessionID {
		t.Errorf("saveSessionID = %q, want %q", receipt.SaveSessionID, saveSessionID)
	}
	if receipt.SaveRevision != saveRevision {
		t.Errorf("saveRevision = %q, want %q", receipt.SaveRevision, saveRevision)
	}
	want, err := saveengine.ChangedScopesForMutationKind(operationKind)
	if err != nil {
		t.Fatalf("ChangedScopesForMutationKind(%q): %v", operationKind, err)
	}
	if !reflect.DeepEqual(receipt.ChangedScopes, want) {
		t.Errorf("changedScopes = %v, want exactly %v", receipt.ChangedScopes, want)
	}
}
