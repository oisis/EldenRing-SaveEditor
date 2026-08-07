package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateResourceAcceptsTypedLegacyLinks(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	resource.Item.Links = schema.ItemLinks{
		AboutTutorialID: knownFact(p, uint32(2010)),
		RelatedEventFlags: []schema.RelatedEventFlag{
			{
				Kind:        knownFact(p, schema.RelatedEventFlagWhetblade),
				EventFlagID: knownFact(p, uint32(65600)),
			},
			{
				Kind:        knownFact(p, schema.RelatedEventFlagAoWMenu),
				EventFlagID: knownFact(p, uint32(65800)),
			},
		},
		RelatedItems: []schema.RelatedItem{{
			Kind:   knownFact(p, schema.RelatedItemBundledAcquisition),
			GameID: knownFact(p, uint32(0x8000C418)),
		}},
		MapFragment: &schema.MapFragmentMetadata{
			Name:           knownFact(p, "Limgrave, West"),
			Area:           knownFact(p, "Limgrave"),
			AcquiredFlagID: knownFact(p, uint32(63010)),
		},
	}

	if err := schema.ValidateResource(resource, sources); err != nil {
		t.Fatalf("ValidateResource: %v", err)
	}
}

func TestValidateResourceRejectsRelatedFlagAlreadyStoredAsUnlock(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	resource.Item.Unlocks = []schema.ItemUnlock{{
		Kind: knownFact(p, "whetblade"), EventFlagID: knownFact(p, uint32(60130)),
	}}
	resource.Item.Links.RelatedEventFlags = []schema.RelatedEventFlag{{
		Kind:        knownFact(p, schema.RelatedEventFlagWhetblade),
		EventFlagID: knownFact(p, uint32(60130)),
	}}

	err := schema.ValidateResource(resource, sources)
	if err == nil || !strings.Contains(err.Error(), "duplicate event flag ID") {
		t.Fatalf("ValidateResource error = %v, want duplicate flag rejection", err)
	}
}

func TestValidateResourceAcceptsRelatedTechnicalAppearanceRecord(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	resource.Item.RelatedTechnicalRecords = []schema.RelatedTechnicalRecord{{
		Kind:   knownFact(p, schema.TechnicalRecordAppearanceState),
		GameID: knownFact(p, uint32(0x400000B6)),
		Description: schema.ItemDescriptionRecord{
			Description: knownFact(p, "Technical appearance-state record"),
		},
		GameMaxInventory: knownFact(p, uint32(999)),
		GameMaxStorage:   knownFact(p, uint32(999)),
		SourceRecords: []schema.ParameterRecord{{
			Table: "EquipParamGoods", RowID: 182, Provenance: p,
			Fields: []schema.ParameterField{{Name: "appearanceReplaceItemId"}},
		}},
	}}

	if err := schema.ValidateResource(resource, sources); err != nil {
		t.Fatalf("ValidateResource: %v", err)
	}
}
