package migration

import "testing"

// TestPreOrderSafetyDerivedFromMergedGestureFlags proves the pre_order flag on a
// slot-only gesture reaches safety.preOrder through the same ItemData/AllGestures
// merge buildSafety already performs. The Ring (Pre-Order) carries pre_order on
// its gesture slot only, so this fails if safety stops merging gesture flags.
func TestPreOrderSafetyDerivedFromMergedGestureFlags(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	theRingPreOrder := findGeneratedItem(t, catalog, 0x40002359)
	if !theRingPreOrder.Safety.PreOrder.Known || !theRingPreOrder.Safety.PreOrder.Value {
		t.Fatalf("The Ring (Pre-Order) safety.preOrder = %#v, want known true", theRingPreOrder.Safety.PreOrder)
	}
}

// TestSafetyDerivedFlagsStrippedFromTopLevelFlags proves the four safety-derived
// tokens no longer persist on any generated top-level item.flags, while the
// safety facts still record them and unrelated flags survive. Piquebone Arrow
// authors ["dlc", "stackable"]; only "stackable" may remain at top level.
func TestSafetyDerivedFlagsStrippedFromTopLevelFlags(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	arrow := findGeneratedItem(t, catalog, 0x03032DE0)
	if got := arrow.Flags.Value; len(got) != 1 || got[0] != "stackable" {
		t.Fatalf("Piquebone Arrow flags = %#v, want [stackable]", got)
	}
	if !arrow.Safety.DLC.Known || !arrow.Safety.DLC.Value {
		t.Fatalf("Piquebone Arrow safety.dlc = %#v, want known true", arrow.Safety.DLC)
	}

	stripped := map[string]struct{}{
		"dlc":            {},
		"no_database":    {},
		"scales_with_ng": {},
		"pre_order":      {},
	}
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil {
			continue
		}
		for _, flag := range item.Flags.Value {
			if _, forbidden := stripped[flag]; forbidden {
				t.Fatalf("item 0x%08X top-level flags retain safety-derived token %q", item.GameID.Value, flag)
			}
		}
	}
}
