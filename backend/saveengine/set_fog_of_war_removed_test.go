package saveengine

import (
	"testing"
)

// The confirmed field, restated literally instead of reused from the
// implementation, so a changed bound fails here: 0x087E to an inclusive 0x10B0
// behind the unlocked-regions list, which is 2099 bytes.
const (
	fogTestFieldStart = int64(0x087E)
	fogTestFieldSize  = int64(0x10B0 - 0x087E + 1)
)

// fogTestFieldAt restates the walk to the field independently of the engine: the
// region count offset of the synthetic fixture, its declared list and the
// confirmed distance to the first Fog of War byte.
func fogTestFieldAt(content gestureTestFixture, regions int) int64 {
	var slotBase int64
	if content.platform == PlatformPS4 {
		slotBase = gestureTestPS4SlotDataBase + int64(content.slot)*gestureTestPS4SlotStride
	} else {
		slotBase = gestureTestPCSlotDataBase + int64(content.slot)*gestureTestPCSlotStride
	}
	return slotBase + regionsTestCountAt(content) + 4 +
		int64(regions)*regionsTestRecordSize + fogTestFieldStart
}

// The dynamic region list sits in front of the field, so a fixture with regions
// proves the offset is measured and not assumed, and both platform bases are
// covered by the same slot content.
func TestSetFogOfWarRemovedFillsExactlyTheFieldOnBothPlatforms(t *testing.T) {
	regions := []uint32{6100000, 6101000, 6102000}
	for _, content := range []gestureTestFixture{
		{platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x01A7},
		{platform: PlatformPS4, slot: 7, flag: 1, anchorAt: 0x1F4C2, projectiles: 37},
	} {
		t.Run(string(content.platform), func(t *testing.T) {
			path := writeRegionsFixture(t, content, regions, uint32(len(regions)))
			fieldAt := fogTestFieldAt(content, len(regions))
			engine := New()
			loaded, err := engine.LoadSave(path, string(content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			snapshot := engine.sessions[loaded.SaveSessionID].snapshot
			before := append([]byte(nil), snapshot.data...)

			result, err := engine.SetFogOfWarRemoved(loaded.SaveSessionID, content.slot, true, "0")
			if err != nil {
				t.Fatalf("SetFogOfWarRemoved: %v", err)
			}
			want := SetFogOfWarRemovedResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   content.slot,
				Removed:       true,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// The zeroed fixture turns every written byte into a changed byte, so
			// the changed set is the written range exactly: no neighbouring byte
			// of the horse and bloodstain prefix or of MenuProfile may appear.
			changed := changedSnapshotBytes(before, snapshot.data)
			if int64(len(changed)) != fogTestFieldSize {
				t.Fatalf("changed %d bytes, want %d", len(changed), fogTestFieldSize)
			}
			if int64(changed[0]) != fieldAt || int64(changed[len(changed)-1]) != fieldAt+fogTestFieldSize-1 {
				t.Fatalf("changed range = 0x%X..0x%X, want 0x%X..0x%X",
					changed[0], changed[len(changed)-1], fieldAt, fieldAt+fogTestFieldSize-1)
			}
			field, err := snapshot.readAt(fieldAt, int(fogTestFieldSize))
			if err != nil {
				t.Fatalf("read field: %v", err)
			}
			for index, value := range field {
				if value != 0xFF {
					t.Fatalf("field byte %d = 0x%02X, want 0xFF", index, value)
				}
			}

			// Repeating the operation is a no-op on the bytes and a normal commit
			// on the session, exactly as the legacy in-place fill was.
			repeated := append([]byte(nil), snapshot.data...)
			second, err := engine.SetFogOfWarRemoved(loaded.SaveSessionID, content.slot, true, "1")
			if err != nil {
				t.Fatalf("SetFogOfWarRemoved repeated: %v", err)
			}
			if second.SaveRevision != "2" {
				t.Errorf("repeated saveRevision = %q, want \"2\"", second.SaveRevision)
			}
			if again := changedSnapshotBytes(repeated, snapshot.data); len(again) != 0 {
				t.Errorf("repeated call changed %d bytes, want 0", len(again))
			}

			if undo := engine.sessions[loaded.SaveSessionID].session.undo; undo != nil {
				t.Errorf("unchanged repeat recorded an undo point: %+v", undo)
			}
		})
	}
}

// The undo point has to carry this operation's own identifier, so an undo can be
// attributed to the mutation that created it.
func TestSetFogOfWarRemovedRecordsItsOwnUndoPoint(t *testing.T) {
	content := gestureTestFixture{platform: PlatformPC, slot: 3, flag: 1, anchorAt: 0x640}
	path := writeRegionsFixture(t, content, []uint32{6100000}, 1)
	engine := New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := engine.SetFogOfWarRemoved(loaded.SaveSessionID, content.slot, true, "0"); err != nil {
		t.Fatalf("SetFogOfWarRemoved: %v", err)
	}
	undo := engine.sessions[loaded.SaveSessionID].session.undo
	if undo == nil {
		t.Fatal("no undo point was recorded")
	}
	if undo.operationKind != "set_fog_of_war_removed" || undo.characterID != content.slot {
		t.Fatalf("undo point = %q/%d, want set_fog_of_war_removed/%d",
			undo.operationKind, undo.characterID, content.slot)
	}
}

func TestSetFogOfWarRemovedRejectsWithoutMutating(t *testing.T) {
	active := gestureTestFixture{platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x640}
	inactive := gestureTestFixture{platform: PlatformPC, slot: 2, anchorAt: 0x640}

	load := func(t *testing.T, path string) (*Engine, string) {
		t.Helper()
		engine := New()
		loaded, err := engine.LoadSave(path, "pc", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return engine, loaded.SaveSessionID
	}

	cases := []struct {
		name      string
		path      func(t *testing.T) string
		removed   bool
		revision  string
		wantError string
	}{
		{
			name:      "removed false",
			path:      func(t *testing.T) string { return writeRegionsFixture(t, active, nil, 0) },
			removed:   false,
			revision:  "0",
			wantError: "removed must be true; restoring Fog of War has no confirmed contract",
		},
		{
			name:      "non canonical revision",
			path:      func(t *testing.T) string { return writeRegionsFixture(t, active, nil, 0) },
			removed:   true,
			revision:  "00",
			wantError: `expectedRevision must be a canonical decimal saveRevision; got "00"`,
		},
		{
			name:      "stale revision",
			path:      func(t *testing.T) string { return writeRegionsFixture(t, active, nil, 0) },
			removed:   true,
			revision:  "7",
			wantError: `expectedRevision "7" does not match the current saveRevision "0"`,
		},
		{
			name:      "inactive character",
			path:      func(t *testing.T) string { return writeRegionsFixture(t, inactive, nil, 0) },
			removed:   true,
			revision:  "0",
			wantError: "character 2 is not active",
		},
		{
			name: "corrupt region count",
			path: func(t *testing.T) string {
				return writeRegionsFixture(t, active, nil, regionsTestMaxCount+1)
			},
			removed:   true,
			revision:  "0",
			wantError: "character 2 declares 20001 unlocked regions, want at most 20000",
		},
		{
			name: "region list outside the slot",
			path: func(t *testing.T) string {
				content := active
				content.anchorAt = gestureTestSlotDataSize - regionsTestCountAt(gestureTestFixture{}) - 4
				return writeRegionsFixture(t, content, nil, 1)
			},
			removed:   true,
			revision:  "0",
			wantError: "unlocked regions of character 2 do not fit into their slot",
		},
		{
			// The list still ends exactly on the slot boundary, so only the field
			// itself reaches past it: the bound the region locator cannot catch.
			name: "field outside the slot",
			path: func(t *testing.T) string {
				content := active
				content.anchorAt = gestureTestSlotDataSize -
					regionsTestCountAt(gestureTestFixture{}) - 4 - regionsTestRecordSize
				return writeRegionsFixture(t, content, []uint32{6100000}, 1)
			},
			removed:   true,
			revision:  "0",
			wantError: "Fog of War bitfield of character 2 does not fit into its slot",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine, sessionID := load(t, testCase.path(t))
			snapshot := engine.sessions[sessionID].snapshot
			before := append([]byte(nil), snapshot.data...)

			result, err := engine.SetFogOfWarRemoved(sessionID, 2, testCase.removed, testCase.revision)
			if err == nil || err.Error() != testCase.wantError {
				t.Fatalf("error = %v, want %q", err, testCase.wantError)
			}
			if result != (SetFogOfWarRemovedResult{}) {
				t.Errorf("result = %+v, want zero value", result)
			}
			if changed := changedSnapshotBytes(before, snapshot.data); len(changed) != 0 {
				t.Errorf("rejected call changed %d bytes, want 0", len(changed))
			}
			info, err := engine.GetSessionInfo(sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			session := engine.sessions[sessionID].session
			if session.revisionString() != "0" || info.UnsavedChanges || session.undo != nil {
				t.Errorf("session after rejection = revision %q, unsaved %v, undo %v; want revision 0, clean and no undo point",
					session.revisionString(), info.UnsavedChanges, session.undo)
			}
		})
	}
}
