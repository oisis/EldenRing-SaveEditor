package migration

import "testing"

func TestReviewedArmorSaveForgeValuesAreDiscarded(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	perfumerRobe := findGeneratedItem(t, catalog, perfumerRobeAlteredItemID).Armor
	if perfumerRobe.Focus.Value != 63 || perfumerRobe.FocusSFV != nil {
		t.Fatalf("Perfumer Robe (Altered) focus = %+v", perfumerRobe.Focus)
	}
}
