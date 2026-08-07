package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestAbsentRequiredContainerIsNotApplicable(t *testing.T) {
	acquisition := buildAcquisition(seed{}, true)
	assertNotApplicableFact(
		t,
		"required container",
		acquisition.RequiredContainerID.Known,
		acquisition.RequiredContainerID.Value == 0,
		acquisition.RequiredContainerID.Provenance,
	)

}

func TestApplicableOptionalMetadataStaysUnresolved(t *testing.T) {
	acquisition := buildAcquisition(seed{}, false).RequiredContainerID
	if acquisition.Provenance.Source == "" || acquisition.Provenance.Method == "" {
		t.Fatalf("required container provenance = %#v, want non-empty source and method", acquisition.Provenance)
	}
	if acquisition.Provenance.MarksNotApplicable() {
		t.Fatalf("required container provenance = %#v, want a plain unknown method", acquisition.Provenance)
	}
}

func TestRequiredContainerNotApplicableScope(t *testing.T) {
	if !requiredContainerIsNotApplicable(
		seed{Name: fireKnightGreatswordName},
		schema.ItemFamilyWeapon,
	) {
		t.Fatal("Fire Knight's Greatsword must be not applicable")
	}
	if requiredContainerIsNotApplicable(
		seed{Name: "Fire Knight's Shortsword"},
		schema.ItemFamilyWeapon,
	) {
		t.Fatal("Fire Knight's Shortsword must remain unresolved")
	}
}

func assertNotApplicableFact(
	t *testing.T,
	name string,
	known bool,
	zeroValue bool,
	provenance schema.Provenance,
) {
	t.Helper()
	if known || !zeroValue {
		t.Fatalf(
			"%s must stay unknown with a zero value, known=%t zeroValue=%t",
			name,
			known,
			zeroValue,
		)
	}
	if provenance.Source == "" || provenance.Method == "" {
		t.Fatalf("%s provenance = %#v, want non-empty source and method", name, provenance)
	}
	if !provenance.MarksNotApplicable() {
		t.Fatalf("%s provenance = %#v, want a not-applicable method", name, provenance)
	}
}
