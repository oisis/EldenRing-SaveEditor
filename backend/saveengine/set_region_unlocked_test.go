package saveengine

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
)

func TestSetRegionUnlockedPCAndPS4(t *testing.T) {
	tests := []struct {
		name        string
		platform    Platform
		slot        int
		initial     []uint32
		regionID    uint32
		unlocked    bool
		wantRegions []uint32
	}{
		{
			name:        "PC add absent regionID",
			platform:    PlatformPC,
			slot:        0,
			initial:     []uint32{6100000, 0, 9999999, 6100000},
			regionID:    6101000,
			unlocked:    true,
			wantRegions: []uint32{6100000, 0, 9999999, 6100000, 6101000},
		},
		{
			name:        "PS4 remove all occurrences of target regionID",
			platform:    PlatformPS4,
			slot:        7,
			initial:     []uint32{6100000, 0, 6101000, 9999999, 6101000, 6102000},
			regionID:    6101000,
			unlocked:    false,
			wantRegions: []uint32{6100000, 0, 9999999, 6102000},
		},
		{
			name:        "PC remove non-existent regionID keeps exact list",
			platform:    PlatformPC,
			slot:        2,
			initial:     []uint32{6100000, 0, 9999999, 6100000},
			regionID:    6105000,
			unlocked:    false,
			wantRegions: []uint32{6100000, 0, 9999999, 6100000},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := rebuildFixtureSlot{
				slot:         tc.slot,
				active:       true,
				anchorAt:     0x01A7,
				regions:      tc.initial,
				menuSize:     0x20,
				tutorialSize: 0x20,
				worldSizes:   [5]uint32{0x10, 0x10, 0x10, 0x10, 0x10},
			}
			path, _, loaded := writeRebuildFixture(t, tc.platform, cfg)
			engine := New()
			info, err := engine.LoadSave(path, string(tc.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			// Capture fixed DLC and fixed Hash before mutation.
			slotBase, _ := eventFlagSlotBounds(tc.platform, tc.slot)
			dlcBefore, err := loaded.snapshot.readAt(slotBase+slotFixedDlcOffset, int(slotFixedDlcSize))
			if err != nil {
				t.Fatalf("read fixed DLC before: %v", err)
			}
			hashBefore, err := loaded.snapshot.readAt(slotBase+slotFixedHashOffset, int(slotFixedHashSize))
			if err != nil {
				t.Fatalf("read fixed Hash before: %v", err)
			}

			result, err := engine.SetRegionUnlocked(info.SaveSessionID, tc.slot, tc.regionID, tc.unlocked, "0")
			if err != nil {
				t.Fatalf("SetRegionUnlocked: %v", err)
			}
			want := SetRegionUnlockedResult{
				SaveSessionID: info.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   tc.slot,
				Unlocked:      tc.unlocked,
			}
			if result != want {
				t.Fatalf("result = %+v, want %+v", result, want)
			}

			// Verify re-read regions.
			characterRegions, err := engine.GetRegions(info.SaveSessionID, tc.slot)
			if err != nil {
				t.Fatalf("GetRegions after mutation: %v", err)
			}
			if !slices.Equal(characterRegions.RegionIDs, tc.wantRegions) {
				t.Fatalf("regions = %v, want %v", characterRegions.RegionIDs, tc.wantRegions)
			}

			// Verify event flags section is valid and readable after rebuild.
			sess := engine.sessions[info.SaveSessionID]
			flagSectionAt, err := eventFlagSectionStart(sess, tc.slot)
			if err != nil {
				t.Fatalf("eventFlagSectionStart after rebuild: %v", err)
			}
			rawFlags, err := sess.snapshot.readAt(flagSectionAt, 1)
			if err != nil || rawFlags[0] != 0xAA {
				t.Fatalf("event flag byte at %d = %v (err %v), want 0xAA", flagSectionAt, rawFlags, err)
			}

			// Verify fixed DLC and fixed Hash blocks are unchanged.
			dlcAfter, err := sess.snapshot.readAt(slotBase+slotFixedDlcOffset, int(slotFixedDlcSize))
			if err != nil {
				t.Fatalf("read fixed DLC after: %v", err)
			}
			if !bytes.Equal(dlcBefore, dlcAfter) {
				t.Fatal("fixed DLC section was altered by SetRegionUnlocked")
			}
			hashAfter, err := sess.snapshot.readAt(slotBase+slotFixedHashOffset, int(slotFixedHashSize))
			if err != nil {
				t.Fatalf("read fixed Hash after: %v", err)
			}
			if !bytes.Equal(hashBefore, hashAfter) {
				t.Fatal("fixed Hash section was altered by SetRegionUnlocked")
			}
		})
	}
}

func TestSetRegionUnlockedIdempotent(t *testing.T) {
	initial := []uint32{6100000, 6101000, 6100000}
	cfg := rebuildFixtureSlot{
		slot:         0,
		active:       true,
		anchorAt:     0x01A7,
		regions:      initial,
		menuSize:     0x20,
		tutorialSize: 0x20,
		worldSizes:   [5]uint32{0x10, 0x10, 0x10, 0x10, 0x10},
	}
	path, _, _ := writeRebuildFixture(t, PlatformPC, cfg)
	engine := New()
	info, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	beforeSnapshot := bytes.Clone(engine.sessions[info.SaveSessionID].snapshot.data)
	engine.mutex.Unlock()

	// Idempotent unlock of existing 6100000.
	res, err := engine.SetRegionUnlocked(info.SaveSessionID, 0, 6100000, true, "0")
	if err != nil {
		t.Fatalf("SetRegionUnlocked idempotent: %v", err)
	}
	if res.SaveRevision != "1" || !res.Unlocked {
		t.Fatalf("idempotent result = %+v", res)
	}

	engine.mutex.Lock()
	session := engine.sessions[info.SaveSessionID]
	if !bytes.Equal(session.snapshot.data, beforeSnapshot) {
		t.Fatal("idempotent mutation modified snapshot bytes")
	}
	if session.session.revisionString() != "1" {
		t.Fatalf("revision = %q, want 1", session.session.revisionString())
	}
	if !session.session.dirty {
		t.Fatal("expected dirty=true after idempotent mutation")
	}
	engine.mutex.Unlock()

	// Revision advanced to 1, dirty is true, but since slot was unchanged, undo point is empty.
	undoState, err := engine.GetUndoState(info.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undoState.Available {
		t.Fatal("idempotent mutation created an undo point for an unchanged slot")
	}

	// Verify list is preserved byte-for-byte including duplicate.
	gotRegions, err := engine.GetRegions(info.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	if !reflect.DeepEqual(gotRegions.RegionIDs, initial) {
		t.Fatalf("regions = %v, want %v", gotRegions.RegionIDs, initial)
	}
}

func TestSetRegionUnlockedRejectsInvalidAndInactiveRequests(t *testing.T) {
	cfg := rebuildFixtureSlot{
		slot:         0,
		active:       true,
		anchorAt:     0x01A7,
		regions:      []uint32{6100000},
		menuSize:     0x20,
		tutorialSize: 0x20,
		worldSizes:   [5]uint32{0x10, 0x10, 0x10, 0x10, 0x10},
	}
	path, _, _ := writeRebuildFixture(t, PlatformPC, cfg)
	engine := New()
	info, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	sess := engine.sessions[info.SaveSessionID]
	beforeSnapshot := bytes.Clone(sess.snapshot.data)
	beforeRev := sess.session.revision
	beforeDirty := sess.session.dirty
	beforeUndo := sess.session.undo
	beforeOwnedSeq := sess.session.ownedSeq
	beforeOwnedByIDCount := len(sess.session.ownedByID)
	beforeOwnedByLocCount := len(sess.session.ownedByLocator)
	engine.mutex.Unlock()

	// Non-canonical revision.
	if _, err := engine.SetRegionUnlocked(info.SaveSessionID, 0, 6101000, true, "01"); err == nil {
		t.Fatal("expected error for non-canonical revision")
	}
	// Stale revision.
	if _, err := engine.SetRegionUnlocked(info.SaveSessionID, 0, 6101000, true, "5"); err == nil {
		t.Fatal("expected error for stale revision")
	}
	// Out-of-range characterID.
	if _, err := engine.SetRegionUnlocked(info.SaveSessionID, 10, 6101000, true, "0"); err == nil {
		t.Fatal("expected error for out of range characterID")
	}
	// Inactive character slot (slot 1 is not active in fixture).
	if _, err := engine.SetRegionUnlocked(info.SaveSessionID, 1, 6101000, true, "0"); err == nil {
		t.Fatal("expected error for inactive character slot")
	}

	engine.mutex.Lock()
	afterSess := engine.sessions[info.SaveSessionID]
	if !bytes.Equal(afterSess.snapshot.data, beforeSnapshot) {
		t.Fatal("snapshot modified on rejected calls")
	}
	if afterSess.session.revision != beforeRev {
		t.Fatalf("revision = %d, want %d", afterSess.session.revision, beforeRev)
	}
	if afterSess.session.dirty != beforeDirty {
		t.Fatalf("dirty = %t, want %t", afterSess.session.dirty, beforeDirty)
	}
	if afterSess.session.undo != beforeUndo {
		t.Fatal("undo modified on rejected calls")
	}
	if afterSess.session.ownedSeq != beforeOwnedSeq {
		t.Fatalf("ownedSeq = %d, want %d", afterSess.session.ownedSeq, beforeOwnedSeq)
	}
	if len(afterSess.session.ownedByID) != beforeOwnedByIDCount {
		t.Fatalf("ownedByID count = %d, want %d", len(afterSess.session.ownedByID), beforeOwnedByIDCount)
	}
	if len(afterSess.session.ownedByLocator) != beforeOwnedByLocCount {
		t.Fatalf("ownedByLocator count = %d, want %d", len(afterSess.session.ownedByLocator), beforeOwnedByLocCount)
	}
	engine.mutex.Unlock()

	// Confirm session revision remains 0 and unchanged.
	gotRegions, err := engine.GetRegions(info.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	if !slices.Equal(gotRegions.RegionIDs, []uint32{6100000}) {
		t.Fatalf("regions altered on failed calls: %v", gotRegions.RegionIDs)
	}
}
