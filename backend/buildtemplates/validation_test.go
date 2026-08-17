package buildtemplates_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
)

func uint32Ptr(v uint32) *uint32 { return &v }
func uint8Ptr(v uint8) *uint8    { return &v }
func intPtr(v int) *int          { return &v }
func strPtr(v string) *string    { return &v }

func TestDecodeTemplate_V1Valid(t *testing.T) {
	rawV1 := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-05-17T10:00:00Z",
  "appVersion": "1.5.8",
  "metadata": {
    "name": "V1 Template",
    "description": "Legacy build template",
    "tags": ["v1", "test"]
  },
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [
        {
          "baseItemID": 1000000,
          "name": "Dagger",
          "category": "melee_armaments",
          "quantity": 1,
          "upgrade": 25,
          "infusionName": "Heavy",
          "container": "inventory",
          "position": 0
        }
      ],
      "storageItems": [
        {
          "baseItemID": 2000000,
          "name": "Buckler",
          "category": "shields",
          "quantity": 1,
          "container": "storage",
          "position": 0
        }
      ]
    }
  }
}`

	tpl, err := buildtemplates.DecodeTemplate([]byte(rawV1))
	if err != nil {
		t.Fatalf("DecodeTemplate v1 failed: %v", err)
	}
	if tpl.Version != 1 {
		t.Errorf("version = %d, want 1", tpl.Version)
	}
	if tpl.Sections.InventoryWorkspace == nil {
		t.Fatal("inventory.workspace is nil")
	}
	if len(tpl.Sections.InventoryWorkspace.InventoryItems) != 1 {
		t.Errorf("inventory items count = %d, want 1", len(tpl.Sections.InventoryWorkspace.InventoryItems))
	}
}

func TestDecodeTemplate_V2Canonical_RoundTripFidelity(t *testing.T) {
	rawV2 := `{
  "schema": "saveforge.build-template",
  "version": 2,
  "createdAt": "2026-08-17T12:00:00Z",
  "appVersion": "2.0.0",
  "metadata": {
    "name": "Full V2 Build",
    "description": "Canonical V2 build with all sections",
    "author": "Tarnished",
    "tags": ["pvp", "meta"],
    "sourceCharacterIndex": 0,
    "sourceCharacterName": "Hero"
  },
  "selection": {
    "profile": {
      "name": true,
      "level": true,
      "runes": true,
      "soulMemory": true,
      "class": true,
      "clearCount": true,
      "scadutreeBlessing": true,
      "shadowRealmBlessing": true,
      "talismanSlots": true
    },
    "stats": true,
    "equipment": {
      "weaponRightHand1": true,
      "weaponLeftHand1": true,
      "talisman1": true
    },
    "spells": {
      "spell1": true,
      "spell2": true
    },
    "items": true,
    "inventoryLayout": true,
    "storageLayout": true
  },
  "sections": {
    "profile": {
      "name": "Hero",
      "level": 125,
      "runes": 50000,
      "soulMemory": 50000,
      "class": "Vagabond",
      "clearCount": 1,
      "scadutreeBlessing": 10,
      "shadowRealmBlessing": 5,
      "talismanSlots": 3
    },
    "stats": {
      "vigor": 60,
      "mind": 20,
      "endurance": 30,
      "strength": 54,
      "dexterity": 18,
      "intelligence": 9,
      "faith": 15,
      "arcane": 11
    },
    "equipment": {
      "weaponRightHand1": {
        "baseItemID": 1000000,
        "name": "Claymore",
        "upgrade": 25,
        "infusionName": "Heavy",
        "aowItemID": 100
      },
      "weaponLeftHand1": {
        "baseItemID": 0
      },
      "talisman1": {
        "baseItemID": 2000,
        "name": "Crimson Amber Medallion +2"
      }
    },
    "spells": {
      "spell1": {
        "baseItemID": 1073747824,
        "name": "Catch Flame"
      },
      "spell2": {
        "baseItemID": 0
      }
    },
    "items": {
      "entries": [
        {
          "entryID": "item-claymore",
          "itemID": 1000000,
          "name": "Claymore",
          "category": "melee_armaments",
          "quantity": 1,
          "location": "inventory",
          "upgradeKind": "standard",
          "upgradeLevel": 25,
          "infusionName": "Heavy",
          "ashOfWarItemID": 100
        },
        {
          "entryID": "item-somber-wpn",
          "itemID": 2000000,
          "name": "Somber Weapon",
          "category": "melee_armaments",
          "quantity": 1,
          "location": "both",
          "upgradeKind": "somber",
          "upgradeLevel": 10
        },
        {
          "entryID": "item-talisman",
          "itemID": 3000000,
          "name": "Talisman",
          "category": "talismans",
          "quantity": 1,
          "location": "inventory",
          "upgradeKind": "none"
        }
      ]
    },
    "inventoryLayout": {
      "entries": [
        {
          "entryRef": "item-claymore",
          "position": 0
        },
        {
          "entryRef": "item-somber-wpn",
          "position": 1
        }
      ]
    },
    "storageLayout": {
      "entries": [
        {
          "entryRef": "item-somber-wpn",
          "position": 0
        }
      ]
    }
  },
  "applyOptions": {
    "items": {
      "mode": "merge",
      "preserveExtraItems": true
    },
    "inventoryLayout": {
      "mode": "replace"
    },
    "storageLayout": {
      "mode": "ignore"
    },
    "weaponLevelOverride": {
      "useTemplateLevels": false,
      "standardOverride": 25,
      "somberOverride": 10
    }
  }
}`

	tpl1, err := buildtemplates.DecodeTemplate([]byte(rawV2))
	if err != nil {
		t.Fatalf("DecodeTemplate v2 failed: %v", err)
	}

	// Verify specific v2 fields are typed and not lost or renamed
	if tpl1.Sections.Spells.Spell1 == nil || tpl1.Sections.Spells.Spell1.BaseItemID != 0x40001770 {
		t.Fatalf("spell1 baseItemID = %v, want 0x40001770", tpl1.Sections.Spells.Spell1)
	}
	if tpl1.Sections.Spells.Spell2 == nil || tpl1.Sections.Spells.Spell2.BaseItemID != 0 {
		t.Fatalf("spell2 baseItemID = %v, want 0", tpl1.Sections.Spells.Spell2)
	}

	item0 := tpl1.Sections.Items.Entries[0]
	if item0.EntryID != "item-claymore" || item0.ItemID != 1000000 || item0.UpgradeLevel == nil || *item0.UpgradeLevel != 25 || item0.AshOfWarItemID == nil || *item0.AshOfWarItemID != 100 {
		t.Fatalf("item0 field mismatch: %+v", item0)
	}

	if tpl1.Sections.InventoryLayout.Entries[0].EntryRef != "item-claymore" {
		t.Fatalf("inventoryLayout entryRef = %q, want item-claymore", tpl1.Sections.InventoryLayout.Entries[0].EntryRef)
	}

	if tpl1.ApplyOptions.Items.Mode != "merge" || !tpl1.ApplyOptions.Items.PreserveExtraItems {
		t.Fatalf("applyOptions.items mismatch: %+v", tpl1.ApplyOptions.Items)
	}
	if tpl1.ApplyOptions.WeaponLevelOverride.UseTemplateLevels || *tpl1.ApplyOptions.WeaponLevelOverride.StandardOverride != 25 {
		t.Fatalf("weaponLevelOverride mismatch: %+v", tpl1.ApplyOptions.WeaponLevelOverride)
	}

	// Re-encode to JSON and decode again
	reEncoded, err := json.Marshal(tpl1)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	tpl2, err := buildtemplates.DecodeTemplate(reEncoded)
	if err != nil {
		t.Fatalf("DecodeTemplate after re-encode failed: %v", err)
	}

	if !reflect.DeepEqual(tpl1, tpl2) {
		t.Fatalf("round-trip deep equality failed:\ntpl1: %+v\ntpl2: %+v", tpl1, tpl2)
	}
}

func TestDecodeTemplate_FailClosedUnknownFields(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{
			name: "unknown top-level field",
			json: `{"schema":"saveforge.build-template","version":1,"createdAt":"2026-05-17T10:00:00Z","extraUnknown":123,"sections":{"inventory.workspace":{"inventoryItems":[{"baseItemID":1,"quantity":1,"container":"inventory","position":0}],"storageItems":[]}}}`,
		},
		{
			name: "unknown section field in template sections",
			json: `{"schema":"saveforge.build-template","version":2,"createdAt":"2026-05-17T10:00:00Z","selection":{"items":true},"sections":{"items":{"entries":[{"entryID":"e1","itemID":1,"category":"melee_armaments","quantity":1,"location":"inventory"}]},"unknownSec":{}}}`,
		},
		{
			name: "unknown field inside item entry",
			json: `{"schema":"saveforge.build-template","version":2,"createdAt":"2026-05-17T10:00:00Z","selection":{"items":true},"sections":{"items":{"entries":[{"entryID":"e1","itemID":1,"category":"melee_armaments","quantity":1,"location":"inventory","badField":"drop"}]}}}`,
		},
		{
			name: "unknown field inside equipment slot ref",
			json: `{"schema":"saveforge.build-template","version":2,"createdAt":"2026-05-17T10:00:00Z","selection":{"equipment":true},"sections":{"equipment":{"weaponRightHand1":{"baseItemID":1,"extraParam":42}}}}`,
		},
		{
			name: "unknown field inside spells section",
			json: `{"schema":"saveforge.build-template","version":2,"createdAt":"2026-05-17T10:00:00Z","selection":{"spells":true},"sections":{"spells":{"spell15":{"baseItemID":1073747824}}}}`,
		},
		{
			name: "unknown field inside applyOptions",
			json: `{"schema":"saveforge.build-template","version":2,"createdAt":"2026-05-17T10:00:00Z","selection":{"items":true},"sections":{"items":{"entries":[{"entryID":"e1","itemID":1,"category":"melee_armaments","quantity":1,"location":"inventory"}]}},"applyOptions":{"unknownOpt":true}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildtemplates.DecodeTemplate([]byte(tc.json))
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("expected an unknown-field decoder error, got: %v", err)
			}
		})
	}
}

func TestDecodeTemplate_RejectsTrailingJSONValue(t *testing.T) {
	raw := `{"schema":"saveforge.build-template","version":1,"sections":{"inventory.workspace":{"inventoryItems":[{"baseItemID":1,"quantity":1,"container":"inventory","position":0}],"storageItems":[]}}} {}`
	if _, err := buildtemplates.DecodeTemplate([]byte(raw)); err == nil || !strings.Contains(err.Error(), "extra data") {
		t.Fatalf("expected trailing JSON error, got %v", err)
	}
}

func TestValidateTemplate_V1KeepsLegacyVersionDispatch(t *testing.T) {
	tpl := buildtemplates.BuildTemplate{
		Schema:       buildtemplates.SchemaKey,
		Version:      1,
		Selection:    &buildtemplates.TemplateSelection{Profile: &buildtemplates.SectionSelection{All: true}},
		ApplyOptions: &buildtemplates.ApplyOptions{Items: &buildtemplates.ItemApplyOptions{Mode: "merge"}},
		Sections: buildtemplates.TemplateSections{
			InventoryWorkspace: &buildtemplates.InventoryWorkspaceSection{
				InventoryItems: []buildtemplates.TemplateItem{{BaseItemID: 1, Quantity: 1, Container: "inventory"}},
				StorageItems:   []buildtemplates.TemplateItem{},
			},
			Profile: &buildtemplates.ProfileSection{},
		},
	}
	if err := buildtemplates.ValidateTemplate(&tpl); err != nil {
		t.Fatalf("v1 validation diverged from v1.5.8/v1.6.8 version dispatch: %v", err)
	}
}

func TestValidateTemplate_ValidationNegativeCases(t *testing.T) {
	cases := []struct {
		name        string
		tpl         buildtemplates.BuildTemplate
		errContains string
	}{
		{
			name: "wrong schema key",
			tpl: buildtemplates.BuildTemplate{
				Schema:    "wrong.schema",
				Version:   1,
				CreatedAt: "2026-05-17T10:00:00Z",
			},
			errContains: "wrong schema",
		},
		{
			name: "unsupported version 3",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   3,
				CreatedAt: "2026-05-17T10:00:00Z",
			},
			errContains: "unsupported version",
		},
		{
			name: "v1 empty workspace",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   1,
				CreatedAt: "2026-05-17T10:00:00Z",
				Sections: buildtemplates.TemplateSections{
					InventoryWorkspace: &buildtemplates.InventoryWorkspaceSection{
						InventoryItems: []buildtemplates.TemplateItem{},
						StorageItems:   []buildtemplates.TemplateItem{},
					},
				},
			},
			errContains: "inventory.workspace is empty",
		},
		{
			name: "v1 item baseItemID 0",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   1,
				CreatedAt: "2026-05-17T10:00:00Z",
				Sections: buildtemplates.TemplateSections{
					InventoryWorkspace: &buildtemplates.InventoryWorkspaceSection{
						InventoryItems: []buildtemplates.TemplateItem{{BaseItemID: 0, Quantity: 1, Container: "inventory", Position: 0}},
						StorageItems:   []buildtemplates.TemplateItem{},
					},
				},
			},
			errContains: "baseItemID=0",
		},
		{
			name: "v1 item wrong container",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   1,
				CreatedAt: "2026-05-17T10:00:00Z",
				Sections: buildtemplates.TemplateSections{
					InventoryWorkspace: &buildtemplates.InventoryWorkspaceSection{
						InventoryItems: []buildtemplates.TemplateItem{{BaseItemID: 1, Quantity: 1, Container: "storage", Position: 0}},
						StorageItems:   []buildtemplates.TemplateItem{},
					},
				},
			},
			errContains: "container=\"storage\" does not match section",
		},
		{
			name: "v2 missing selection",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
			},
			errContains: "v2 template requires a selection object",
		},
		{
			name: "v2 selection has no fields",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{},
			},
			errContains: "selection has no selected fields",
		},
		{
			name: "v2 selection.profile unknown field",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{
					Profile: &buildtemplates.SectionSelection{Fields: map[string]bool{"unknownField": true}},
				},
				Sections: buildtemplates.TemplateSections{
					Profile: &buildtemplates.ProfileSection{},
				},
			},
			errContains: "selection.profile has unknown field",
		},
		{
			name: "v2 selection.stats unknown field",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{
					Stats: &buildtemplates.SectionSelection{Fields: map[string]bool{"unknownStat": true}},
				},
				Sections: buildtemplates.TemplateSections{
					Stats: &buildtemplates.StatsSection{},
				},
			},
			errContains: "selection.stats has unknown field",
		},
		{
			name: "v2 selection.equipment unknown slot",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{
					Equipment: &buildtemplates.SectionSelection{Fields: map[string]bool{"unknownSlot": true}},
				},
				Sections: buildtemplates.TemplateSections{
					Equipment: &buildtemplates.EquipmentSection{},
				},
			},
			errContains: "selection.equipment has unknown slot",
		},
		{
			name: "v2 selection.spells unknown slot",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{
					Spells: &buildtemplates.SectionSelection{Fields: map[string]bool{"spell15": true}},
				},
				Sections: buildtemplates.TemplateSections{
					Spells: &buildtemplates.SpellsSection{},
				},
			},
			errContains: "selection.spells has unknown slot",
		},
		{
			name: "v2 selection.items with field map rejected",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{
					Items: &buildtemplates.SectionSelection{Fields: map[string]bool{"anything": true}},
				},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{},
				},
			},
			errContains: "selection.items accepts only a boolean",
		},
		{
			name: "v2 selection without section",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{
					Profile: &buildtemplates.SectionSelection{All: true},
				},
			},
			errContains: "selection.profile is selected but sections.profile is missing",
		},
		{
			name: "v2 profile name exceeds UTF-16 cap",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Profile: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Profile: &buildtemplates.ProfileSection{
						Name: strPtr("This name is way too long for UTF-16 cap"),
					},
				},
			},
			errContains: "profile.name exceeds 16 UTF-16 code units",
		},
		{
			name: "v2 profile level out of range",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Profile: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Profile: &buildtemplates.ProfileSection{
						Level: uint32Ptr(714),
					},
				},
			},
			errContains: "profile.level=714 out of range",
		},
		{
			name: "v2 stats out of range",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Stats: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Stats: &buildtemplates.StatsSection{
						Vigor: uint32Ptr(100),
					},
				},
			},
			errContains: "stats.vigor=100 out of range",
		},
		{
			name: "v2 equipment upgrade out of range",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Equipment: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Equipment: &buildtemplates.EquipmentSection{
						WeaponRightHand1: &buildtemplates.EquipmentItemRef{
							BaseItemID: 1000,
							Upgrade:    intPtr(26),
						},
					},
				},
			},
			errContains: "equipment.weaponRightHand1.upgrade=26 out of range",
		},
		{
			name: "v2 equipment aowItemID 0",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Equipment: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Equipment: &buildtemplates.EquipmentSection{
						WeaponRightHand1: &buildtemplates.EquipmentItemRef{
							BaseItemID: 1000,
							AoWItemID:  uint32Ptr(0),
						},
					},
				},
			},
			errContains: "equipment.weaponRightHand1.aowItemID=0 is invalid",
		},
		{
			name: "v2 spell prefix wrong",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Spells: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Spells: &buildtemplates.SpellsSection{
						Spell1: &buildtemplates.SpellSlotRef{
							BaseItemID: 0x10000000,
						},
					},
				},
			},
			errContains: "spells.spell1.baseItemID=0x10000000 has wrong prefix",
		},
		{
			name: "v2 items duplicate entryID",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "dup-id", ItemID: 1, Category: "melee_armaments", Quantity: 1, Location: "inventory"},
							{EntryID: "dup-id", ItemID: 2, Category: "shields", Quantity: 1, Location: "inventory"},
						},
					},
				},
			},
			errContains: "entryID \"dup-id\" already used",
		},
		{
			name: "v2 items unknown category",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "unknown_cat", Quantity: 1, Location: "inventory"},
						},
					},
				},
			},
			errContains: "unknown category \"unknown_cat\"",
		},
		{
			name: "v2 items quantity 0",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "melee_armaments", Quantity: 0, Location: "inventory"},
						},
					},
				},
			},
			errContains: "quantity=0 not allowed",
		},
		{
			name: "v2 items somber upgrade > 10",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "melee_armaments", Quantity: 1, Location: "inventory", UpgradeKind: "somber", UpgradeLevel: uint8Ptr(11)},
						},
					},
				},
			},
			errContains: "upgradeLevel=11 out of range [0, 10] for upgradeKind=somber",
		},
		{
			name: "v2 items non-upgradable with upgradeLevel",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "talismans", Quantity: 1, Location: "inventory", UpgradeKind: "none", UpgradeLevel: uint8Ptr(1)},
						},
					},
				},
			},
			errContains: "upgradeLevel=1 set but upgradeKind=\"none\"",
		},
		{
			name: "v2 layout entryRef not in items",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{InventoryLayout: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "melee_armaments", Quantity: 1, Location: "inventory"},
						},
					},
					InventoryLayout: &buildtemplates.InventoryLayoutSection{
						Entries: []buildtemplates.LayoutEntry{
							{EntryRef: "unmatched-ref", Position: 0},
						},
					},
				},
			},
			errContains: "entryRef \"unmatched-ref\" does not match any items.entries.entryID",
		},
		{
			name: "v2 layout duplicate position",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{InventoryLayout: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "melee_armaments", Quantity: 1, Location: "inventory"},
							{EntryID: "e2", ItemID: 2, Category: "shields", Quantity: 1, Location: "inventory"},
						},
					},
					InventoryLayout: &buildtemplates.InventoryLayoutSection{
						Entries: []buildtemplates.LayoutEntry{
							{EntryRef: "e1", Position: 0},
							{EntryRef: "e2", Position: 0},
						},
					},
				},
			},
			errContains: "position=0 already used",
		},
		{
			name: "v2 applyOptions invalid mode",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "melee_armaments", Quantity: 1, Location: "inventory"},
						},
					},
				},
				ApplyOptions: &buildtemplates.ApplyOptions{
					Items: &buildtemplates.ItemApplyOptions{Mode: "invalidMode"},
				},
			},
			errContains: "applyOptions.items.mode=\"invalidMode\" is invalid",
		},
		{
			name: "v2 applyOptions weapon override useTemplateLevels with override values",
			tpl: buildtemplates.BuildTemplate{
				Schema:    buildtemplates.SchemaKey,
				Version:   2,
				CreatedAt: "2026-05-17T10:00:00Z",
				Selection: &buildtemplates.TemplateSelection{Items: &buildtemplates.SectionSelection{All: true}},
				Sections: buildtemplates.TemplateSections{
					Items: &buildtemplates.ItemsSection{
						Entries: []buildtemplates.TemplateItemEntryV2{
							{EntryID: "e1", ItemID: 1, Category: "melee_armaments", Quantity: 1, Location: "inventory"},
						},
					},
				},
				ApplyOptions: &buildtemplates.ApplyOptions{
					WeaponLevelOverride: &buildtemplates.WeaponLevelOverride{
						UseTemplateLevels: true,
						StandardOverride:  uint8Ptr(25),
					},
				},
			},
			errContains: "useTemplateLevels=true is mutually exclusive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := buildtemplates.ValidateTemplate(&tc.tpl)
			if err == nil {
				t.Fatalf("expected validation error containing %q, got nil", tc.errContains)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("expected error containing %q, got %q", tc.errContains, err.Error())
			}
		})
	}
}
