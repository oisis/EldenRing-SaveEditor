package core

import "testing"

// A save always carries ten slot structures. The ones for characters that were
// never created are zero-filled, which every range check in the diagnostics
// would otherwise read as corruption: Level 0 and all eight attributes 0 make
// nine criticals plus one stats_formula warning per unused slot. Real saves are
// normally far from full, so without the guard a healthy file reports dozens of
// critical issues it does not have.
func TestDiagnoseSaveCorruption_UnusedSlotReportsNothing(t *testing.T) {
	slot := &SaveSlot{Version: 0, Data: make([]byte, SlotSize)}

	diag := DiagnoseSaveCorruption(slot, 7)

	if diag.SlotIndex != 7 {
		t.Fatalf("SlotIndex = %d, want 7", diag.SlotIndex)
	}
	if len(diag.Issues) != 0 {
		t.Fatalf("unused slot reported %d issue(s), want 0: %+v", len(diag.Issues), diag.Issues)
	}
}

// Boundary partner of the test above: the guard must key on Version alone, not
// on the data being zero-filled. The very same buffer, marked as a slot that a
// save version actually wrote, must still be diagnosed — otherwise the guard
// would silence real corruption in an active character.
func TestDiagnoseSaveCorruption_UsedSlotStillReportsBadStats(t *testing.T) {
	slot := &SaveSlot{Version: 1, Data: make([]byte, SlotSize)}

	diag := DiagnoseSaveCorruption(slot, 7)

	stats := 0
	for _, issue := range diag.Issues {
		if issue.Category == "stats" && issue.Severity == SeverityCritical {
			stats++
		}
	}
	if stats == 0 {
		t.Fatalf("used slot with Level 0 reported no critical stats issue: %+v", diag.Issues)
	}
}
