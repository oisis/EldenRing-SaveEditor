package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
)

// The Item Database is the one general item list of the application, so what it
// may show is a profile decision and never a client one. This file covers that
// public boundary; which limits a profile applies is covered by
// backend/safetyprofile.

// itemDatabaseCatalog rebuilds the prototype catalog with the safety facts of
// the one item resource rewritten, so a test states the fact it is about
// instead of depending on what the shipped document happens to declare.
func itemDatabaseCatalog(t *testing.T, banRisk, cutContent bool) (*gamecatalog.Catalog, string) {
	t.Helper()

	manifest, resources, err := prototype.Load()
	if err != nil {
		t.Fatalf("prototype.Load: %v", err)
	}
	key := ""
	for index, resource := range resources {
		if resource.Kind != schema.ResourceKindItem || resource.Item == nil {
			continue
		}
		document := *resource.Item
		document.Safety.BanRisk = schema.Fact[bool]{
			Known:      true,
			Value:      banRisk,
			Provenance: document.Safety.BanRisk.Provenance,
		}
		document.Safety.CutContent = schema.Fact[bool]{
			Known:      true,
			Value:      cutContent,
			Provenance: document.Safety.CutContent.Provenance,
		}
		resources[index].Item = &document
		key = resource.Key
		break
	}
	if key == "" {
		t.Fatal("the prototype catalog holds no item resource")
	}
	gameCatalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog, key
}

func itemDatabaseKeys(result catalog.GetItemDatabaseResult) []string {
	keys := make([]string, 0, len(result.Resources))
	for _, entry := range result.Resources {
		keys = append(keys, entry.Key)
	}
	return keys
}

// A ban-risk item is reachable under Chaos and under no other profile, and the
// answer always names the profile it was resolved under.
func TestGetItemDatabaseHidesBanRiskOutsideChaos(t *testing.T) {
	gameCatalog, banRiskKey := itemDatabaseCatalog(t, true, false)

	for _, profile := range []safetyprofile.Profile{
		safetyprofile.Safe, safetyprofile.ExpandedLimits, safetyprofile.Chaos,
	} {
		t.Run(string(profile), func(t *testing.T) {
			result, err := catalog.GetItemDatabase(gameCatalog, string(profile), "", "", "",
				false, nil, catalog.GetItemDatabaseSortCatalog, 0, 0)
			if err != nil {
				t.Fatalf("GetItemDatabase: %v", err)
			}
			if result.SafetyProfile != string(profile) {
				t.Errorf("safetyProfile = %q, want %q", result.SafetyProfile, profile)
			}
			visible := false
			for _, key := range itemDatabaseKeys(result) {
				if key == banRiskKey {
					visible = true
				}
			}
			if want := profile == safetyprofile.Chaos; visible != want {
				t.Errorf("the ban-risk item is visible = %v under %q, want %v; keys = %v",
					visible, profile, want, itemDatabaseKeys(result))
			}
			if result.Total != len(result.Resources) {
				t.Errorf("total = %d with %d resources on one page",
					result.Total, len(result.Resources))
			}
		})
	}
}

// An unknown profile is refused rather than silently replaced by the default,
// so a caller can never widen what the list shows by sending an unknown value.
func TestGetItemDatabaseRejectsAnUnknownProfile(t *testing.T) {
	gameCatalog, _ := itemDatabaseCatalog(t, false, false)

	result, err := catalog.GetItemDatabase(gameCatalog, "chaos_mode", "", "", "",
		false, nil, catalog.GetItemDatabaseSortCatalog, 0, 0)
	if err == nil {
		t.Fatalf("an unknown profile was accepted: %+v", result)
	}
	if len(result.Resources) != 0 || result.Total != 0 {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
