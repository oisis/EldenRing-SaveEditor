package migration

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateVerifiesAllAshOfWarCompatibilityBits(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	count := 0
	upperBits := 0
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyAshOfWar {
			continue
		}
		count++
		mask := resource.Item.AshOfWar.CompatibilityMask
		if !mask.Known ||
			mask.Provenance.Source != sourceRegulationEquipParamGemRaw {
			t.Fatalf(
				"Ash of War 0x%08X compatibility = %#v",
				resource.Item.GameID.Value,
				mask,
			)
		}
		if mask.Value>>40 != 0 {
			upperBits++
		}
	}
	if count != 116 {
		t.Fatalf("Ash of War documents = %d, want 116", count)
	}
	if upperBits == 0 {
		t.Fatal("no Ash of War exercises DLC compatibility bits 40..43")
	}
}

func TestRegulationParameterReaderUsesLogicalProvenance(t *testing.T) {
	data := readLocalRegulationParameterFixture(t)
	source := data.manifestSource()
	if source.Location != "regulation.bin/params/EquipParamGem.param" ||
		source.ID != sourceRegulationEquipParamGemRaw ||
		source.Evidence != schema.EvidenceRegulation ||
		!source.Reviewed ||
		len(source.Version) != 64 {
		t.Fatalf("raw parameter source = %#v", source)
	}
	if strings.Contains(source.Location, "tmp/") {
		t.Fatalf("raw parameter source exposes local path %q", source.Location)
	}
}

func TestRegulationParameterReaderRejectsTruncatedInput(t *testing.T) {
	_, err := readRegulationParameterFS(fstest.MapFS{
		equipParamGemFilename: &fstest.MapFile{Data: []byte("short")},
	})
	if err == nil || !strings.Contains(err.Error(), "truncated PARAM header") {
		t.Fatalf("readRegulationParameterFS error = %v", err)
	}
}
