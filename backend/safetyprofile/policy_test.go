package safetyprofile

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The policy is the single place the three profiles are interpreted, so this
// test states the whole agreed contract: which limit applies, what stays
// hidden, and what a mutation is allowed to write.

func knownUint32(value uint32) schema.Fact[uint32] {
	return schema.Fact[uint32]{Known: true, Value: value}
}

func knownBool(value bool) schema.Fact[bool] {
	return schema.Fact[bool]{Known: true, Value: value}
}

// item builds a document with base limits and, optionally, the two Safe Mode
// fields. A nil Safe Mode field is the catalog stating no narrower rule.
func item(safeInventory, safeStorage *uint32) *schema.ItemDocument {
	document := &schema.ItemDocument{}
	document.Storage.MaxInventory = knownUint32(600)
	document.Storage.MaxStorage = knownUint32(500)
	if safeInventory != nil {
		fact := knownUint32(*safeInventory)
		document.Storage.SafeModeMaxInventory = &fact
	}
	if safeStorage != nil {
		fact := knownUint32(*safeStorage)
		document.Storage.SafeModeMaxStorage = &fact
	}
	return document
}

func TestLimitsFollowTheProfile(t *testing.T) {
	t.Parallel()

	safeInventory := uint32(99)
	withSafeMode := item(&safeInventory, nil)

	cases := []struct {
		profile           Profile
		document          *schema.ItemDocument
		wantInventory     uint32
		wantStorage       uint32
		wantInventoryKnwn bool
	}{
		// Safe uses the Safe Mode field where the item declares one, and falls
		// back to the base limit where it does not. It never falls back to zero.
		{Safe, withSafeMode, 99, 500, true},
		{ExpandedLimits, withSafeMode, 600, 500, true},
		{Chaos, withSafeMode, 600, 500, true},
		{Safe, item(nil, nil), 600, 500, true},
	}
	for _, testCase := range cases {
		inventory, known := InventoryLimit(testCase.profile, testCase.document)
		if known != testCase.wantInventoryKnwn || inventory != testCase.wantInventory {
			t.Errorf("InventoryLimit(%q) = %d, %t; want %d, %t",
				testCase.profile, inventory, known, testCase.wantInventory, testCase.wantInventoryKnwn)
		}
		storage, known := StorageLimit(testCase.profile, testCase.document)
		if !known || storage != testCase.wantStorage {
			t.Errorf("StorageLimit(%q) = %d, %t; want %d, true",
				testCase.profile, storage, known, testCase.wantStorage)
		}
	}
}

// A catalog that states no limit is reported as unknown rather than as zero, so
// the caller rejects the operation instead of writing against a default.
func TestAnUnknownLimitStaysUnknown(t *testing.T) {
	t.Parallel()

	empty := &schema.ItemDocument{}
	if _, known := InventoryLimit(Safe, empty); known {
		t.Error("InventoryLimit reported a limit for an item that declares none")
	}
	if _, known := StorageLimit(Chaos, empty); known {
		t.Error("StorageLimit reported a limit for an item that declares none")
	}
}

func TestVisibilityFollowsTheProfileExceptForNoDatabase(t *testing.T) {
	t.Parallel()

	banRisk := &schema.ItemDocument{}
	banRisk.Safety.BanRisk = knownBool(true)
	cutContent := &schema.ItemDocument{}
	cutContent.Safety.CutContent = knownBool(true)
	noDatabase := &schema.ItemDocument{}
	noDatabase.Safety.NoDatabase = knownBool(true)
	presentation := &schema.ItemDocument{}
	presentation.Safety.DLC = knownBool(true)
	presentation.Safety.PreOrder = knownBool(true)
	// An undecided flag is not a true one: unknown catalog data never hides a row.
	undecided := &schema.ItemDocument{}
	undecided.Safety.BanRisk = schema.Fact[bool]{Known: false, Value: true}

	cases := []struct {
		name     string
		document *schema.ItemDocument
		safe     bool
		expanded bool
		chaos    bool
	}{
		{"banRisk", banRisk, true, true, false},
		{"cutContent", cutContent, true, true, false},
		{"noDatabase", noDatabase, true, true, true},
		{"dlc and preOrder", presentation, false, false, false},
		{"undecided banRisk", undecided, false, false, false},
	}
	for _, testCase := range cases {
		for profile, want := range map[Profile]bool{
			Safe: testCase.safe, ExpandedLimits: testCase.expanded, Chaos: testCase.chaos,
		} {
			if got := HiddenFromItemDatabase(profile, testCase.document); got != want {
				t.Errorf("HiddenFromItemDatabase(%q, %s) = %t, want %t",
					profile, testCase.name, got, want)
			}
		}
	}
}

func TestAllowMutationRefusesWhatTheProfileForbids(t *testing.T) {
	t.Parallel()

	banRisk := &schema.ItemDocument{}
	banRisk.Safety.BanRisk = knownBool(true)
	cutContent := &schema.ItemDocument{}
	cutContent.Safety.CutContent = knownBool(true)
	ordinary := &schema.ItemDocument{}
	// noDatabase hides a resource from the general list but stays writable by the
	// feature that owns it, so it is never a mutation refusal.
	noDatabase := &schema.ItemDocument{}
	noDatabase.Safety.NoDatabase = knownBool(true)

	if err := AllowMutation(Safe, banRisk, true); err == nil {
		t.Error("Safe accepted a ban-risk resource even with a confirmation")
	}
	if err := AllowMutation(ExpandedLimits, cutContent, true); err == nil {
		t.Error("ExpandedLimits accepted a cut-content resource")
	}
	if err := AllowMutation(Chaos, banRisk, false); err == nil {
		t.Error("Chaos accepted a ban-risk resource without a confirmation")
	}
	if err := AllowMutation(Chaos, banRisk, true); err != nil {
		t.Errorf("Chaos refused a confirmed ban-risk resource: %v", err)
	}
	if err := AllowMutation(Safe, ordinary, false); err != nil {
		t.Errorf("Safe refused an ordinary resource: %v", err)
	}
	if err := AllowMutation(Safe, noDatabase, false); err != nil {
		t.Errorf("Safe refused a noDatabase resource: %v", err)
	}
	if err := AllowMutation(Safe, nil, true); err == nil {
		t.Error("AllowMutation accepted a resource without an item document")
	}
}

func TestParseAcceptsExactlyTheThreeProfiles(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"safe", "expanded_limits", "chaos"} {
		if _, err := Parse(value); err != nil {
			t.Errorf("Parse(%q): %v", value, err)
		}
	}
	// There is deliberately no empty form, no alias and no case-insensitive
	// match: a caller states which profile it means.
	for _, value := range []string{"", " safe", "Safe", "SAFE", "chaos_mode", "expanded"} {
		if _, err := Parse(value); err == nil {
			t.Errorf("Parse(%q) was accepted", value)
		}
	}
	if Default != Safe {
		t.Errorf("Default = %q, want %q", Default, Safe)
	}
}
