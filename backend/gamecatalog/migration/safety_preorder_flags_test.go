package migration

import (
	"encoding/json"
	"testing"
)

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

// TestFlagsFieldRemovedFromSerializedDocuments proves the retired top-level
// item.flags and variants[].data.flags fields no longer serialize on any
// generated document, while the raw gesture.slots[].flags field is preserved.
func TestFlagsFieldRemovedFromSerializedDocuments(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	hasFlagsKey := func(v any) bool {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var decoded map[string]json.RawMessage
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		_, present := decoded["flags"]
		return present
	}

	variantsChecked := 0
	gestureSlotFlagsSeen := false
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil {
			continue
		}
		if hasFlagsKey(item) {
			t.Fatalf("item 0x%08X serialized a top-level flags field", item.GameID.Value)
		}
		for variantIndex := range item.Variants {
			variantsChecked++
			if hasFlagsKey(item.Variants[variantIndex].Data) {
				t.Fatalf("item 0x%08X variant %d serialized a data.flags field", item.GameID.Value, variantIndex)
			}
		}
		if item.Gesture != nil {
			for _, slot := range item.Gesture.Slots {
				if hasFlagsKey(slot) {
					gestureSlotFlagsSeen = true
				}
			}
		}
	}

	if variantsChecked == 0 {
		t.Fatal("no variant documents exercised; expected variant coverage")
	}
	if !gestureSlotFlagsSeen {
		t.Fatal("gesture.slots[].flags field was not preserved on any gesture item")
	}
}
