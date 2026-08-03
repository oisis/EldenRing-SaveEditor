package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateResourceAllowsUnknownOptionalFacts(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	dagger := resources[0]
	dagger.Item.Presentation.Description.Known = false
	dagger.Item.Presentation.Description.Value = ""
	dagger.Item.Presentation.IconPath.Known = false
	dagger.Item.Presentation.IconPath.Value = ""
	dagger.Item.Storage.RecordMode.Known = false
	dagger.Item.Storage.RecordMode.Value = ""
	dagger.Item.Weapon.WeaponTypeID.Known = false
	dagger.Item.Weapon.WeaponTypeID.Value = 0

	if err := schema.ValidateResource(dagger, sources); err != nil {
		t.Fatalf("ValidateResource with unknown optional facts: %v", err)
	}
}

func TestValidateResourceRejectsUnknownCapabilityWithRules(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	dagger := resources[0]
	rules := schema.StackRules{MaxPerStack: 1}
	dagger.Item.Capabilities.Stack.Known = false
	dagger.Item.Capabilities.Stack.Enabled = true
	dagger.Item.Capabilities.Stack.Rules = &rules

	err := schema.ValidateResource(dagger, sources)
	if err == nil || !strings.Contains(err.Error(), "unknown capability") {
		t.Fatalf("ValidateResource error = %v, want unknown capability rejection", err)
	}
}

func TestValidateResourceRejectsEnabledCapabilityWithoutRules(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	dagger := resources[0]
	dagger.Item.Capabilities.Upgrade.Rules = nil

	err := schema.ValidateResource(dagger, sources)
	if err == nil || !strings.Contains(err.Error(), "enabled capability requires rules") {
		t.Fatalf("ValidateResource error = %v, want missing rules rejection", err)
	}
}

func TestValidateResourceRejectsDisabledCapabilityWithRules(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	determination := resources[1]
	rules := schema.UpgradeRules{
		Model:    schema.UpgradeModelStandard,
		MaxLevel: 25,
	}
	determination.Item.Capabilities.Upgrade.Rules = &rules

	err := schema.ValidateResource(determination, sources)
	if err == nil || !strings.Contains(err.Error(), "disabled capability cannot contain rules") {
		t.Fatalf("ValidateResource error = %v, want disabled capability rejection", err)
	}
}

func TestValidateResourceRejectsDuplicateAffinity(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	dagger := resources[0]
	dagger.Item.Capabilities.Infusion.Rules.AllowedAffinities = append(
		dagger.Item.Capabilities.Infusion.Rules.AllowedAffinities,
		schema.AffinityStandard,
	)

	err := schema.ValidateResource(dagger, sources)
	if err == nil || !strings.Contains(err.Error(), "duplicate affinity") {
		t.Fatalf("ValidateResource error = %v, want duplicate affinity rejection", err)
	}
}

func TestValidateResourceRejectsUnknownProvenanceSource(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	dagger := resources[0]
	dagger.Item.Presentation.DisplayName.Provenance.Source = "missing"

	err := schema.ValidateResource(dagger, sources)
	if err == nil || !strings.Contains(err.Error(), "unknown provenance source") {
		t.Fatalf("ValidateResource error = %v, want unknown provenance rejection", err)
	}
}

func TestValidateResourceRejectsVariantProvenanceWithoutItsOwnSourceRecord(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	dagger := resources[0]
	variant := &dagger.Item.Variants[0]
	variant.SourceRecords = variant.SourceRecords[1:]

	err := schema.ValidateResource(dagger, sources)
	if err == nil || !strings.Contains(err.Error(), "not covered by sourceRecords") {
		t.Fatalf(
			"ValidateResource error = %v, want uncovered variant provenance rejection",
			err,
		)
	}
}
