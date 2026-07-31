package migration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGeneratedWeaponVariantsNeverUseGestureFallbacks(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var weaponVariants int
	var regulationOnlyVariants int
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		for _, variant := range resource.Item.Variants {
			weaponVariants++
			if !variant.Data.Capabilities.Upgrade.Known ||
				!variant.Data.Capabilities.Infusion.Known ||
				!variant.Data.Capabilities.AshOfWarMount.Known ||
				!variant.Data.Capabilities.Stack.Known ||
				!variant.Data.Capabilities.Equipment.Known {
				t.Fatalf(
					"weapon variant 0x%08X contains an unknown capability",
					variant.GameID.Value,
				)
			}
			if !variant.Data.Storage.RecordMode.Known ||
				!variant.Data.Storage.MaxInventory.Known ||
				!variant.Data.Storage.MaxStorage.Known {
				t.Fatalf(
					"weapon variant 0x%08X contains unknown authored storage",
					variant.GameID.Value,
				)
			}
			raw, marshalErr := json.Marshal(variant.Data)
			if marshalErr != nil {
				t.Fatalf("marshal variant 0x%08X: %v", variant.GameID.Value, marshalErr)
			}
			text := string(raw)
			if strings.Contains(text, "slot-only gesture") ||
				strings.Contains(text, "AllGestures") {
				t.Fatalf(
					"weapon variant 0x%08X uses gesture fallback provenance",
					variant.GameID.Value,
				)
			}
			if strings.Contains(text, "Regulation-only variant") {
				regulationOnlyVariants++
			}
		}
	}
	if weaponVariants != 2784 {
		t.Fatalf("weapon variants = %d, want 2784", weaponVariants)
	}
	if regulationOnlyVariants != 2596 {
		t.Fatalf(
			"Regulation-only weapon variants = %d, want 2596",
			regulationOnlyVariants,
		)
	}
}

func TestGeneratedSpiritAshVariantsCarryCompleteUpgradeEvidence(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var variants int
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilySpiritAsh {
			continue
		}
		for _, variant := range resource.Item.Variants {
			variants++
			if len(variant.Data.Capabilities.Upgrade.RulesEvidence) != 21 {
				t.Fatalf(
					"Spirit Ash variant 0x%08X rules evidence = %d, want 21",
					variant.GameID.Value,
					len(variant.Data.Capabilities.Upgrade.RulesEvidence),
				)
			}
			if len(variant.SourceRecords) != 21 {
				t.Fatalf(
					"Spirit Ash variant 0x%08X source records = %d, want 21",
					variant.GameID.Value,
					len(variant.SourceRecords),
				)
			}
		}
	}
	if variants != 840 {
		t.Fatalf("Spirit Ash variants = %d, want 840", variants)
	}
}

func TestGeneratedEquipmentCapabilityIsKnownForEveryDocument(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			continue
		}
		if !resource.Item.Capabilities.Equipment.Known {
			t.Fatalf(
				"canonical item 0x%08X has unknown equipment capability",
				resource.Item.GameID.Value,
			)
		}
		for _, variant := range resource.Item.Variants {
			if !variant.Data.Capabilities.Equipment.Known {
				t.Fatalf(
					"variant item 0x%08X has unknown equipment capability",
					variant.GameID.Value,
				)
			}
		}
	}
}
