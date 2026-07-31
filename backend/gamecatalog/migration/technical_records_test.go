package migration

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateAppearanceStateTechnicalRecordsExactCoverage(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want := map[uint32]uint32{
		0x40000064: 0x400000AA,
		0x40000065: 0x400000AB,
		0x40000066: 0x400000AC,
		0x40000068: 0x400000AE,
		0x40000069: 0x400000AF,
		0x4000006C: 0x400000B2,
		0x4000006D: 0x400000B3,
		0x4000006E: 0x400000B4,
		0x4000006F: 0x400000B7,
		0x40000070: 0x400000B8,
		0x40000082: 0x400000B5,
		0x40000096: 0x400000B6,
	}
	assertTechnicalSeedsCoverTargets(t, collectLegacySnapshot(), want)
	got := make(map[uint32]uint32, len(want))
	allItemIDs := make(map[uint32]struct{}, 3838)
	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			continue
		}
		allItemIDs[resource.Item.GameID.Value] = struct{}{}
		for _, alias := range resource.Item.Aliases {
			allItemIDs[alias.GameID.Value] = struct{}{}
		}
		for _, variant := range resource.Item.Variants {
			allItemIDs[variant.GameID.Value] = struct{}{}
		}
		if len(resource.Item.RelatedTechnicalRecords) == 0 {
			continue
		}
		if len(resource.Item.RelatedTechnicalRecords) != 1 {
			t.Fatalf(
				"item 0x%08X related technical records = %d, want 1",
				resource.Item.GameID.Value,
				len(resource.Item.RelatedTechnicalRecords),
			)
		}
		record := resource.Item.RelatedTechnicalRecords[0]
		got[resource.Item.GameID.Value] = record.GameID.Value
		assertAppearanceStateRecord(
			t,
			resource.Item.GameID.Value,
			record,
		)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("appearance-state links = %#v, want %#v", got, want)
	}
	for _, technicalID := range want {
		if _, indexed := allItemIDs[technicalID]; indexed {
			t.Fatalf(
				"technical appearance-state ID 0x%08X was indexed as an item or alias",
				technicalID,
			)
		}
	}
}

func assertTechnicalSeedsCoverTargets(
	t *testing.T,
	snapshot legacySnapshot,
	links map[uint32]uint32,
) {
	t.Helper()
	seeds := make(map[uint32]technicalRecordSeed, len(snapshot.TechnicalRecords))
	for _, record := range snapshot.TechnicalRecords {
		seeds[record.ID] = record
	}
	for _, targetID := range links {
		if _, exists := seeds[targetID]; !exists {
			t.Fatalf(
				"appearance-state target 0x%08X has no complete legacy technical seed",
				targetID,
			)
		}
	}
}

func assertAppearanceStateRecord(
	t *testing.T,
	ownerID uint32,
	record schema.RelatedTechnicalRecord,
) {
	t.Helper()
	if !record.Kind.Known ||
		record.Kind.Value != schema.TechnicalRecordAppearanceState {
		t.Fatalf("item 0x%08X technical kind = %#v", ownerID, record.Kind)
	}
	description, exists := data.Descriptions[record.GameID.Value]
	if !exists {
		t.Fatalf(
			"technical item 0x%08X has no legacy description",
			record.GameID.Value,
		)
	}
	if want := buildDescriptionRecord(*copyLegacyDescription(description)); !reflect.DeepEqual(record.Description, want) {
		t.Fatalf(
			"technical item 0x%08X description differs from legacy source",
			record.GameID.Value,
		)
	}
	limits, exists := data.GameLimitsByItemID[record.GameID.Value]
	if !exists || !limits.InventoryKnown || !limits.StorageKnown {
		t.Fatalf(
			"technical item 0x%08X has incomplete legacy limits",
			record.GameID.Value,
		)
	}
	if !record.GameMaxInventory.Known ||
		record.GameMaxInventory.Value != limits.MaxInventory ||
		!record.GameMaxStorage.Known ||
		record.GameMaxStorage.Value != limits.MaxStorage {
		t.Fatalf(
			"technical item 0x%08X limits = %#v/%#v, want %d/%d",
			record.GameID.Value,
			record.GameMaxInventory,
			record.GameMaxStorage,
			limits.MaxInventory,
			limits.MaxStorage,
		)
	}
	if len(record.SourceRecords) != 2 ||
		record.SourceRecords[0].Table != string(RegulationTableGoods) ||
		record.SourceRecords[0].RowID != int64(ownerID&0x0FFFFFFF) ||
		record.SourceRecords[1].Table != string(RegulationTableGoods) ||
		record.SourceRecords[1].RowID != int64(record.GameID.Value&0x0FFFFFFF) {
		t.Fatalf(
			"technical item 0x%08X source records = %#v",
			record.GameID.Value,
			record.SourceRecords,
		)
	}
}
