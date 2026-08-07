package dbviewer

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestItemTemplateShowsStructuredMetadataSections(t *testing.T) {
	sourceID := schema.SourceID("legacy")
	provenance := schema.Provenance{Source: sourceID, Method: "copied from legacy database"}
	server := &Server{
		sources: map[schema.SourceID]schema.DataSource{
			sourceID: {
				ID:       sourceID,
				Location: "legacy-db",
			},
		},
	}
	templates, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	server.templates = templates

	acquisition := schema.ItemAcquisition{
		RequiredContainerID: schema.Fact[uint32]{
			Known:      true,
			Value:      77,
			Provenance: provenance,
		},
	}
	textMetadata := schema.ItemTextMetadata{
		DescriptionSource: schema.Fact[string]{
			Known:      true,
			Value:      "Descriptions map",
			Provenance: provenance,
		},
	}
	modifiers := schema.ItemModifiers{
		EquipLoad: &schema.EquipLoadModifier{
			EnduranceBonus: schema.Fact[int32]{
				Known:      true,
				Value:      5,
				Provenance: provenance,
			},
			EquipLoadRate: schema.Fact[float64]{
				Known:      true,
				Value:      0.15,
				Provenance: provenance,
			},
		},
	}
	unlocks := []schema.ItemUnlock{{
		Kind: schema.Fact[string]{
			Known:      true,
			Value:      "cookbook",
			Provenance: provenance,
		},
		EventFlagID: schema.Fact[uint32]{
			Known:      true,
			Value:      123,
			Provenance: provenance,
		},
	}}
	links := schema.ItemLinks{
		AboutTutorialID: schema.Fact[uint32]{
			Known:      true,
			Value:      2010,
			Provenance: provenance,
		},
	}
	technical := []schema.RelatedTechnicalRecord{{
		Kind: schema.Fact[schema.TechnicalRecordKind]{
			Known:      true,
			Value:      schema.TechnicalRecordAppearanceState,
			Provenance: provenance,
		},
		GameID: schema.Fact[uint32]{
			Known:      true,
			Value:      0x400000B6,
			Provenance: provenance,
		},
		GameMaxInventory: schema.Fact[uint32]{
			Known:      true,
			Value:      999,
			Provenance: provenance,
		},
	}}

	response := httptest.NewRecorder()
	server.render(response, "item", itemPage{
		Meta:          pageMeta{},
		Name:          "Metadata test",
		GameID:        "0x00000001",
		GameIDPath:    "00000001",
		EntryType:     "Canonical",
		TextMetadata:  server.readableFacts(textMetadata, ""),
		Acquisition:   server.readableFacts(acquisition, ""),
		Modifiers:     server.readableFacts(modifiers, ""),
		Links:         server.readableFacts(links, ""),
		Unlocks:       server.readableFacts(unlocks, "Unlock"),
		TechnicalData: server.readableFacts(technical, "Technical record"),
	})
	if response.Code != 200 {
		t.Fatalf("render status = %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"<h2>Text metadata</h2>",
		"Description source",
		"Descriptions map",
		"<h2>Acquisition</h2>",
		"Required container ID",
		"<h2>Modifiers</h2>",
		"Equip load / Endurance bonus",
		"Equip load / Equip load rate",
		"<h2>Related game data</h2>",
		"About tutorial ID",
		"<h2>Unlocks</h2>",
		"Unlock 1 / Kind",
		"cookbook",
		"<h2>Related technical records</h2>",
		"Technical record 1 / Kind",
		"appearance_state",
		"Technical record 1 / Game max inventory",
		"999",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("item page does not contain %q", expected)
		}
	}
}

func TestOptionalMetadataFactsRenderNotApplicable(t *testing.T) {
	server := &Server{}
	acquisition := server.metadataFacts(
		schema.ItemAcquisition{
			RequiredContainerID: schema.Fact[uint32]{
				Provenance: notApplicableProvenance("never held in a legacy RequiredContainer"),
			},
		},
		"",
		schema.ItemFamilyWeapon,
	)
	if !containsFact(acquisition, "Required container ID", "N/A") {
		t.Fatalf("required container fact = %#v, want N/A", acquisition)
	}
}

func TestUnresolvedOptionalMetadataFactsRenderUnknown(t *testing.T) {
	server := &Server{}
	acquisition := server.metadataFacts(
		schema.ItemAcquisition{
			RequiredContainerID: schema.Fact[uint32]{
				Provenance: schema.Provenance{
					Source: schema.SourceSaveForgeLegacy,
					Method: "legacy RequiredContainer has no entry for this item",
				},
			},
		},
		"",
		schema.ItemFamilySpiritAsh,
	)
	if !containsFact(acquisition, "Required container ID", "Unknown") {
		t.Fatalf("unresolved required container fact = %#v, want Unknown", acquisition)
	}
}
