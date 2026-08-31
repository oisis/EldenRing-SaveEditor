package desktop

import (
	"strings"
	"testing"
)

// TestSaveFileFiltersOfferTheConfirmedContainersAndAllFiles pins the exact
// dialog configuration.
//
// The filter is a convenience and never a validation rule: SaveEngine
// recognises a container from its leading magic, so the "all files" entry has
// to stay reachable or a valid save with an unexpected name would become
// unselectable. The extension list is pinned exactly, because adding one on a
// guess would advertise support this build has no evidence for.
func TestSaveFileFiltersOfferTheConfirmedContainersAndAllFiles(t *testing.T) {
	if len(saveFileFilters) != 2 {
		t.Fatalf("the dialog offers %d filters, want exactly 2", len(saveFileFilters))
	}

	const wantPattern = "*.sl2;*.co2;*.bak;*.dat"
	if saveFileFilters[0].Pattern != wantPattern {
		t.Errorf("save filter pattern = %q, want %q", saveFileFilters[0].Pattern, wantPattern)
	}
	// The confirmed PS4 container has to be selectable by name, not only through
	// the fallback entry.
	if !strings.Contains(saveFileFilters[0].Pattern, "*.dat") {
		t.Error("the save filter does not offer the confirmed PS4 extension *.dat")
	}
	if saveFileFilters[0].DisplayName == "" {
		t.Error("the save filter has no display name")
	}

	fallback := saveFileFilters[len(saveFileFilters)-1]
	if fallback.Pattern != "*.*" {
		t.Fatalf("the last filter pattern = %q, want the all-files fallback %q",
			fallback.Pattern, "*.*")
	}
	if fallback.DisplayName == "" {
		t.Error("the all-files fallback has no display name")
	}
}
