package world

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The capability list is the complete World mutation contract the frontend
// renders its actions from, so the exact set, its order, the backend risk of
// every entry and the single bulk operation are all asserted here.
func TestGetWorldMutationCapabilities(t *testing.T) {
	t.Parallel()

	result, err := GetWorldMutationCapabilities()
	if err != nil {
		t.Fatalf("GetWorldMutationCapabilities: %v", err)
	}

	want := []string{
		"lock_all_spectral_steed_attires",
		"set_bell_bearing_unlocked",
		"set_boss_defeated",
		"set_colosseum_unlocked",
		"set_cookbook_unlocked",
		"set_fog_of_war_removed",
		"set_gesture_unlocked",
		"set_grace_visited",
		"set_map_region_revealed",
		"set_quest_step",
		"set_region_unlocked",
		"set_spectral_steed_attire",
		"set_summoning_pool_activated",
		"set_tutorial_unlocked",
		"set_whetblade_unlocked",
	}
	if len(result.Capabilities) != len(want) {
		t.Fatalf("capability count = %d, want %d", len(result.Capabilities), len(want))
	}
	for index, capability := range result.Capabilities {
		if capability.OperationKind != want[index] {
			t.Errorf("capability %d = %q, want %q", index, capability.OperationKind, want[index])
		}
		// The risk and its reason are the backend's own, taken from the same
		// description the operation history presents.
		described, err := saveengine.DescribeMutationOperation(capability.OperationKind)
		if err != nil {
			t.Fatalf("DescribeMutationOperation(%q): %v", capability.OperationKind, err)
		}
		if capability.Risk != described.Risk || capability.RiskReason != described.RiskReason {
			t.Errorf("capability %q risk = %q/%q, want %q/%q",
				capability.OperationKind, capability.Risk, capability.RiskReason,
				described.Risk, described.RiskReason)
		}
		if capability.Risk != saveengine.OperationRiskWarning {
			t.Errorf("capability %q risk = %q, want %q",
				capability.OperationKind, capability.Risk, saveengine.OperationRiskWarning)
		}
		if capability.RiskReason == "" {
			t.Errorf("capability %q carries no risk reason", capability.OperationKind)
		}
		wantBulk := capability.OperationKind == "lock_all_spectral_steed_attires"
		if capability.SupportsBulk != wantBulk {
			t.Errorf("capability %q supportsBulk = %t, want %t",
				capability.OperationKind, capability.SupportsBulk, wantBulk)
		}
	}
}

// An operation kind SaveEngine does not know must be refused, not published
// with a default risk level.
func TestDescribeMutationOperationRejectsAnUnknownKind(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"", "set_world_flag"} {
		if _, err := saveengine.DescribeMutationOperation(kind); err == nil {
			t.Errorf("DescribeMutationOperation(%q) returned no error", kind)
		} else if kind != "" && !strings.Contains(err.Error(), kind) {
			t.Errorf("DescribeMutationOperation(%q) error = %v, want it to name the kind", kind, err)
		}
	}
}
