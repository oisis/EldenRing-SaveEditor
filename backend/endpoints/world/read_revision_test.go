package world

import "testing"

func TestRequireSameSaveRevisionComparesOpaqueValuesExactly(t *testing.T) {
	t.Parallel()

	if err := requireSameSaveRevision("  Revision 09  ", "  Revision 09  "); err != nil {
		t.Fatalf("requireSameSaveRevision(exact match): %v", err)
	}
	if err := requireSameSaveRevision("9", "09"); err == nil {
		t.Fatal("requireSameSaveRevision(9, 09) succeeded; want exact-comparison error")
	}
	if err := requireSameSaveRevision("Revision 09", "  Revision 09  "); err == nil {
		t.Fatal("requireSameSaveRevision(trimmed, spaced) succeeded; want exact-comparison error")
	}
}
