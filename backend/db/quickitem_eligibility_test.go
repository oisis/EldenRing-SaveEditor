package db

import "testing"

// TestGetQuickItemEligibleItems guards that Quick Items delegate to the exact
// same policy as the Quick pouch: the result must be identical to
// GetPouchEligibleItems (same length, same IDs and names in the same order).
// Inclusion/exclusion cases live in pouch_eligibility_test.go and are not
// duplicated here.
func TestGetQuickItemEligibleItems(t *testing.T) {
	quick := GetQuickItemEligibleItems("PS4")
	pouch := GetPouchEligibleItems("PS4")

	if len(quick) != len(pouch) {
		t.Fatalf("length mismatch: quick=%d pouch=%d", len(quick), len(pouch))
	}
	for i := range quick {
		if quick[i].ID != pouch[i].ID || quick[i].Name != pouch[i].Name {
			t.Errorf("mismatch at %d: quick={0x%08X %q} pouch={0x%08X %q}",
				i, quick[i].ID, quick[i].Name, pouch[i].ID, pouch[i].Name)
		}
	}
}
