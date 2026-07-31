package dbviewer

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestStorageFactsShowsSaveForgeValueNextToRegulationValue(t *testing.T) {
	regulationSource := schema.SourceID("regulation")
	server := &Server{sources: map[schema.SourceID]schema.DataSource{
		regulationSource: {
			ID:       regulationSource,
			Location: "regulation.bin/csv/EquipParamGoods.csv",
		},
		schema.SourceSaveForgeLegacy: {
			ID:       schema.SourceSaveForgeLegacy,
			Location: "backend/db/data",
		},
	}}
	regulation := schema.Provenance{
		Source: regulationSource,
		Method: "test Regulation value",
	}
	saveForge := schema.Fact[uint32]{
		Known: true,
		Value: 1,
		Provenance: schema.Provenance{
			Source: schema.SourceSaveForgeLegacy,
			Method: "test SaveForge value",
		},
	}
	storage := schema.ItemStorage{
		RecordMode:       schema.Fact[schema.RecordMode]{Known: true, Value: schema.RecordModeQuantityStack, Provenance: regulation},
		MaxInventory:     schema.Fact[uint32]{Known: true, Value: 99, Provenance: regulation},
		MaxInventorySFV:  &saveForge,
		MaxStorage:       schema.Fact[uint32]{Known: true, Value: 600, Provenance: regulation},
		GameMaxInventory: schema.Fact[uint32]{Known: true, Value: 99, Provenance: regulation},
		GameMaxStorage:   schema.Fact[uint32]{Known: true, Value: 600, Provenance: regulation},
	}

	facts := server.storageFacts(storage)
	if !containsFact(facts, "Maximum inventory", "99") {
		t.Fatal("Regulation maximum inventory is missing")
	}
	if !containsFact(
		facts,
		"Maximum inventory — SaveForge value",
		"1",
	) {
		t.Fatal("SaveForge maximum inventory is missing")
	}
}
