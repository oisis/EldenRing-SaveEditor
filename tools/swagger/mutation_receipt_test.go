package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/favorites"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// receiptMigratedResults are the mutation result schemas that carry the shared
// MutationReceipt today. Stage 3b.1 migrated the SaveSession and Network batch
// and stage 3b.2 added the Character, Appearance, Templates and Favorites batch.
// Inventory, Equipment, World and Diagnostics still return their old shape, and
// this list must grow only together with their migration.
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
