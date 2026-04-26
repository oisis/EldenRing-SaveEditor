package data

import "testing"

// TestBellBearingItemToFlagIDCoverage verifies the BB item→flag map references
// flags that exist in BellBearings, and that every non-cut-content BB key item
// is mapped.
func TestBellBearingItemToFlagIDCoverage(t *testing.T) {
	for itemID, flagID := range BellBearingItemToFlagID {
		if _, ok := BellBearings[flagID]; !ok {
			t.Errorf("itemID 0x%X maps to flag %d which is not in BellBearings", itemID, flagID)
		}
	}

	// Every non-cut-content BB in key_items must have a flag.
	for itemID, item := range KeyItems {
		if !contains(item.Name, "Bell Bearing") {
			continue
		}
		if hasFlag(item.Flags, "cut_content") {
			continue
		}
		if _, ok := BellBearingItemToFlagID[itemID]; !ok {
			t.Errorf("BB item 0x%X (%q) has no flag mapping", itemID, item.Name)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hasFlag(flags []string, target string) bool {
	for _, f := range flags {
		if f == target {
			return true
		}
	}
	return false
}
