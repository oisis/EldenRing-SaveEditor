package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type rebuildFixtureSlot struct {
	slot         int
	active       bool
	version      uint32
	anchorAt     int64
	regions      []uint32
	menuSize     uint32
	tutorialSize uint32
	worldSizes   [5]uint32
	breakHeader  string
	tailPatches  map[int64]byte
}

func writeRebuildFixture(t *testing.T, platform Platform, cfg rebuildFixtureSlot) (string, []byte, *loadedSave) {
	t.Helper()

	var data []byte
	var userDataBase, slotBase int64
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userDataBase = pcUserData10DataOffset
		slotBase = eventFlagTestPCSlotDataBase + int64(cfg.slot)*eventFlagTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userDataBase = ps4UserData10DataOffset
		slotBase = eventFlagTestPS4SlotDataBase + int64(cfg.slot)*eventFlagTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	if cfg.active {
		data[userDataBase+userData10ActiveFlagsOffset+int64(cfg.slot)] = userData10ActiveFlagValue
	}

	ver := cfg.version
	if ver == 0 && cfg.breakHeader != "version_zero" {
		ver = 230
	}
	binary.LittleEndian.PutUint32(data[slotBase:], ver)

	if cfg.breakHeader != "no_anchor" {
		copy(data[slotBase+cfg.anchorAt:], eventFlagTestAnchor)
	}

	put32 := func(at int64, v uint32) {
		binary.LittleEndian.PutUint32(data[slotBase+at:], v)
	}

	// Projectiles = 0
	projAt := cfg.anchorAt + eventFlagTestProjectileCountAt
	put32(projAt, 0)

	// UnlockedRegions at countAt
	countAt := projAt + 4 + eventFlagTestBlocksBeforeStorage +
		eventFlagTestStorageBoxSize + eventFlagTestGestureSectionSize
	put32(countAt, uint32(len(cfg.regions)))
	for i, id := range cfg.regions {
		put32(countAt+4+int64(i)*4, id)
	}

	// Post-regions sequential walk
	pos := countAt + 4 + int64(len(cfg.regions))*4

	// WorldHead (117 bytes)
	for i := int64(0); i < worldHeadSectionSize; i++ {
		data[slotBase+pos+i] = byte(0x10 + (i % 0x20))
	}
	pos += worldHeadSectionSize

	// MenuProfile: 8-byte header + payload
	if cfg.breakHeader == "broken_menu" {
		put32(pos+4, uint32(eventFlagMaxDynamicSize+1))
	} else {
		put32(pos+4, cfg.menuSize)
	}
	for i := int64(0); i < int64(cfg.menuSize); i++ {
		data[slotBase+pos+8+i] = byte(0x30 + (i % 0x10))
	}
	pos += 8 + int64(cfg.menuSize)

	// TrophyEquip (52 bytes)
	for i := int64(0); i < eventFlagTrophyEquipSize; i++ {
		data[slotBase+pos+i] = byte(0x40 + (i % 0x10))
	}
	pos += eventFlagTrophyEquipSize

	// GaItemGameData (112008 bytes)
	put32(pos, 0x12345678)
	pos += eventFlagGaItemGameDataSize

	// TutorialData: 8-byte header + payload
	if cfg.breakHeader == "broken_tutorial" {
		put32(pos+4, uint32(eventFlagMaxDynamicSize+1))
	} else {
		put32(pos+4, cfg.tutorialSize)
	}
	for i := int64(0); i < int64(cfg.tutorialSize); i++ {
		data[slotBase+pos+8+i] = byte(0x50 + (i % 0x10))
	}
	pos += 8 + int64(cfg.tutorialSize)

	// PreEventFlagsScalars (29 bytes)
	for i := int64(0); i < eventFlagScalarsSize; i++ {
		data[slotBase+pos+i] = byte(0x60 + (i % 0x10))
	}
	pos += eventFlagScalarsSize

	// EventFlags (1833376 bytes)
	data[slotBase+pos] = 0xAA
	data[slotBase+pos+int64(eventFlagSectionSize+eventFlagTerminatorSize)-1] = 0x55
	pos += int64(eventFlagSectionSize + eventFlagTerminatorSize)

	// WorldGeom 5 blocks
	for i, size := range cfg.worldSizes {
		if cfg.breakHeader == "broken_world" && i == 0 {
			put32(pos, uint32(worldBlockLimits[0]))
		} else {
			put32(pos, size)
		}
		for j := int64(0); j < int64(size); j++ {
			data[slotBase+pos+4+j] = byte(0x70 + byte(i)*0x10 + byte(j%10))
		}
		pos += 4 + int64(size)
	}

	// PlayerCoordinates (61 bytes)
	for i := int64(0); i < playerCoordinatesSize; i++ {
		data[slotBase+pos+i] = byte(0x80 + (i % 0x10))
	}
	pos += playerCoordinatesSize

	// SpawnPoint (10 + 4 + 1 = 15 bytes for version 230)
	put32(pos+2, 0x11223344)
	put32(pos+6, 0x55667788)
	put32(pos+10, 0x99AABBCC)
	data[slotBase+pos+14] = 0xDD
	pos += int64(spawnPointFixedSize + 4 + 1)

	// NetMan (4 + 0x20000 bytes)
	put32(pos, 0xCAFEBABE)
	data[slotBase+pos+4] = 0xEE
	data[slotBase+pos+4+0x1FFFF] = 0xFF
	pos += netManSectionSize

	// TrailingFixedBlock (130 bytes)
	for i := int64(0); i < trailingFixedBlockSize; i++ {
		data[slotBase+pos+i] = byte(0x90 + (i % 0x20))
	}
	pos += trailingFixedBlockSize

	// PlayerGameDataHash (128 bytes)
	for i := int64(0); i < playerGameDataHashSize; i++ {
		data[slotBase+pos+i] = byte(0xA0 + (i % 0x20))
	}
	pos += playerGameDataHashSize

	// Fixed End-of-Slot DLC (50 bytes) & Hash (128 bytes)
	for i := int64(0); i < slotFixedDlcSize; i++ {
		data[slotBase+slotFixedDlcOffset+i] = byte(0xB0 + (i % 0x10))
	}
	for i := int64(0); i < slotFixedHashSize; i++ {
		data[slotBase+slotFixedHashOffset+i] = byte(0xC0 + (i % 0x10))
	}

	// Apply custom tail patches relative to slotBase
	for off, val := range cfg.tailPatches {
		data[slotBase+off] = val
	}

	path := filepath.Join(t.TempDir(), "save.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	slotBytes := make([]byte, characterSlotDataSize)
	copy(slotBytes, data[slotBase:slotBase+characterSlotDataSize])

	engine := New()
	info, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	loaded := engine.sessions[info.SaveSessionID]

	return path, slotBytes, loaded
}

func TestRebuildSlotIdentity(t *testing.T) {
	platforms := []Platform{PlatformPC, PlatformPS4}
	initialRegions := []uint32{1001000, 1001001, 1001002, 6100000}

	for _, p := range platforms {
		t.Run(string(p), func(t *testing.T) {
			_, originalSlot, loaded := writeRebuildFixture(t, p, rebuildFixtureSlot{
				slot:         0,
				active:       true,
				version:      230,
				anchorAt:     accountIDTestFirstAnchorAt,
				regions:      initialRegions,
				menuSize:     16,
				tutorialSize: 32,
				worldSizes:   [5]uint32{16, 16, 32, 32, 64},
			})

			rebuilt, err := rebuildSlotWithRegions(loaded, 0, initialRegions)
			if err != nil {
				t.Fatalf("rebuildSlotWithRegions: %v", err)
			}

			if len(rebuilt) != characterSlotDataSize {
				t.Fatalf("len(rebuilt) = 0x%X, want 0x%X", len(rebuilt), characterSlotDataSize)
			}

			if !bytes.Equal(rebuilt, originalSlot) {
				t.Errorf("identity rebuild on %s did not produce byte-for-byte identical output", p)
			}
		})
	}
}

func TestRebuildSlotGrowthPreservesNonZeroMidTailAndPostRegions(t *testing.T) {
	platforms := []Platform{PlatformPC, PlatformPS4}
	initialRegions := []uint32{1001000, 1001001, 1001002}
	newRegions := []uint32{1001000, 1001001, 1001002, 6100000} // growth delta = +4 bytes

	countAt := int64(accountIDTestFirstAnchorAt) + eventFlagTestProjectileCountAt + 4 +
		eventFlagTestBlocksBeforeStorage + eventFlagTestStorageBoxSize + eventFlagTestGestureSectionSize
	origRegsEnd := countAt + 4 + int64(len(initialRegions))*4
	postSectionsLen := int64(worldHeadSectionSize + 8 + 16 + eventFlagTrophyEquipSize +
		eventFlagGaItemGameDataSize + 8 + 32 + eventFlagScalarsSize +
		int64(eventFlagSectionSize+eventFlagTerminatorSize) +
		(4 + 16) + (4 + 16) + (4 + 32) + (4 + 32) + (4 + 64) +
		playerCoordinatesSize + spawnPointFixedSize + 4 + 1 +
		netManSectionSize + trailingFixedBlockSize + playerGameDataHashSize)
	origPostSectionsEnd := origRegsEnd + postSectionsLen

	tailStartOff := origPostSectionsEnd
	tailMidOff := origPostSectionsEnd + 100
	tailDiscardedOff := slotFixedDlcOffset - 2 // non-zero byte in truncated 4-byte suffix

	for _, p := range platforms {
		t.Run(string(p), func(t *testing.T) {
			_, originalSlot, loaded := writeRebuildFixture(t, p, rebuildFixtureSlot{
				slot:         0,
				active:       true,
				version:      230,
				anchorAt:     accountIDTestFirstAnchorAt,
				regions:      initialRegions,
				menuSize:     16,
				tutorialSize: 32,
				worldSizes:   [5]uint32{16, 16, 32, 32, 64},
				tailPatches: map[int64]byte{
					tailStartOff:     0x42,
					tailMidOff:       0x99,
					tailDiscardedOff: 0x88,
				},
			})

			rebuilt, err := rebuildSlotWithRegions(loaded, 0, newRegions)
			if err != nil {
				t.Fatalf("rebuildSlotWithRegions on %s: %v", p, err)
			}

			if len(rebuilt) != characterSlotDataSize {
				t.Fatalf("len(rebuilt) = 0x%X, want 0x%X", len(rebuilt), characterSlotDataSize)
			}

			preRegsLen := countAt
			// 1. Pre-regions match verbatim
			if !bytes.Equal(rebuilt[:preRegsLen], originalSlot[:preRegsLen]) {
				t.Errorf("pre-regions altered during growth on %s", p)
			}

			// 2. UnlockedRegions contains 4 entries
			count := binary.LittleEndian.Uint32(rebuilt[preRegsLen:])
			if count != 4 {
				t.Errorf("region count = %d, want 4 on %s", count, p)
			}

			// 3. Whole post-regions block is preserved and shifted by +4 bytes
			newPostSectionsStart := preRegsLen + 4 + int64(len(newRegions))*4
			newPostSectionsEnd := newPostSectionsStart + postSectionsLen
			if !bytes.Equal(rebuilt[newPostSectionsStart:newPostSectionsEnd],
				originalSlot[origRegsEnd:origPostSectionsEnd]) {
				t.Errorf("entire post-regions block was not preserved verbatim at shifted offset on %s", p)
			}

			// 4. Non-zero bytes at start and middle of tail are preserved at shifted positions (+4)
			if rebuilt[tailStartOff+4] != 0x42 {
				t.Errorf("tail start non-zero byte not preserved on %s, got 0x%02X", p, rebuilt[tailStartOff+4])
			}
			if rebuilt[tailMidOff+4] != 0x99 {
				t.Errorf("tail middle non-zero byte not preserved on %s, got 0x%02X", p, rebuilt[tailMidOff+4])
			}

			// 5. Fixed DLC and Hash match verbatim
			if !bytes.Equal(rebuilt[slotFixedDlcOffset:slotFixedDlcOffset+slotFixedDlcSize],
				originalSlot[slotFixedDlcOffset:slotFixedDlcOffset+slotFixedDlcSize]) {
				t.Errorf("fixed DLC section not preserved at end of slot on %s", p)
			}
			if !bytes.Equal(rebuilt[slotFixedHashOffset:slotFixedHashOffset+slotFixedHashSize],
				originalSlot[slotFixedHashOffset:slotFixedHashOffset+slotFixedHashSize]) {
				t.Errorf("fixed Hash section not preserved at end of slot on %s", p)
			}
		})
	}
}

func TestRebuildSlotShrinkPreservesEntireTailAndZeroesNewSlack(t *testing.T) {
	platforms := []Platform{PlatformPC, PlatformPS4}
	initialRegions := []uint32{1001000, 1001001, 1001002, 6100000}
	newRegions := []uint32{1001000, 1001001, 1001002} // shrink delta = -4 bytes

	countAt := int64(accountIDTestFirstAnchorAt) + eventFlagTestProjectileCountAt + 4 +
		eventFlagTestBlocksBeforeStorage + eventFlagTestStorageBoxSize + eventFlagTestGestureSectionSize
	origRegsEnd := countAt + 4 + int64(len(initialRegions))*4
	postSectionsLen := int64(worldHeadSectionSize + 8 + 16 + eventFlagTrophyEquipSize +
		eventFlagGaItemGameDataSize + 8 + 32 + eventFlagScalarsSize +
		int64(eventFlagSectionSize+eventFlagTerminatorSize) +
		(4 + 16) + (4 + 16) + (4 + 32) + (4 + 32) + (4 + 64) +
		playerCoordinatesSize + spawnPointFixedSize + 4 + 1 +
		netManSectionSize + trailingFixedBlockSize + playerGameDataHashSize)
	origPostSectionsEnd := origRegsEnd + postSectionsLen
	originalTailLen := slotFixedDlcOffset - origPostSectionsEnd

	tailStartOff := origPostSectionsEnd
	tailMidOff := origPostSectionsEnd + 50
	tailNearEndOff := slotFixedDlcOffset - 1 // 1 byte before DLC

	for _, p := range platforms {
		t.Run(string(p), func(t *testing.T) {
			_, originalSlot, loaded := writeRebuildFixture(t, p, rebuildFixtureSlot{
				slot:         0,
				active:       true,
				version:      230,
				anchorAt:     accountIDTestFirstAnchorAt,
				regions:      initialRegions,
				menuSize:     16,
				tutorialSize: 32,
				worldSizes:   [5]uint32{16, 16, 32, 32, 64},
				tailPatches: map[int64]byte{
					tailStartOff:   0x11,
					tailMidOff:     0x22,
					tailNearEndOff: 0x33,
				},
			})

			rebuilt, err := rebuildSlotWithRegions(loaded, 0, newRegions)
			if err != nil {
				t.Fatalf("rebuildSlotWithRegions on %s: %v", p, err)
			}

			if len(rebuilt) != characterSlotDataSize {
				t.Fatalf("len(rebuilt) = 0x%X, want 0x%X", len(rebuilt), characterSlotDataSize)
			}

			preRegsLen := countAt
			// 1. Pre-regions match
			if !bytes.Equal(rebuilt[:preRegsLen], originalSlot[:preRegsLen]) {
				t.Errorf("pre-regions altered during shrink on %s", p)
			}

			// 2. Whole post-regions block shifted by -4 bytes
			newPostSectionsStart := preRegsLen + 4 + int64(len(newRegions))*4
			newPostSectionsEnd := newPostSectionsStart + postSectionsLen
			if !bytes.Equal(rebuilt[newPostSectionsStart:newPostSectionsEnd],
				originalSlot[origRegsEnd:origPostSectionsEnd]) {
				t.Errorf("entire post-regions block was not preserved verbatim at shifted offset on %s", p)
			}

			// 3. Entire originalTail shifted left by 4 bytes and preserved verbatim
			if !bytes.Equal(rebuilt[newPostSectionsEnd:newPostSectionsEnd+originalTailLen],
				originalSlot[origPostSectionsEnd:origPostSectionsEnd+originalTailLen]) {
				t.Errorf("entire original tail was not preserved during shrink on %s", p)
			}

			// 4. The 4 bytes directly preceding fixed DLC are zeroed
			for i := slotFixedDlcOffset - 4; i < slotFixedDlcOffset; i++ {
				if rebuilt[i] != 0 {
					t.Errorf("byte at offset 0x%X before DLC is 0x%02X, want 0x00 on %s", i, rebuilt[i], p)
				}
			}

			// 5. Fixed DLC and Hash match verbatim
			if !bytes.Equal(rebuilt[slotFixedDlcOffset:slotFixedDlcOffset+slotFixedDlcSize],
				originalSlot[slotFixedDlcOffset:slotFixedDlcOffset+slotFixedDlcSize]) {
				t.Errorf("fixed DLC section not preserved at end of slot on %s", p)
			}
			if !bytes.Equal(rebuilt[slotFixedHashOffset:slotFixedHashOffset+slotFixedHashSize],
				originalSlot[slotFixedHashOffset:slotFixedHashOffset+slotFixedHashSize]) {
				t.Errorf("fixed Hash section not preserved at end of slot on %s", p)
			}
		})
	}
}

func TestRebuildSlotPreservesRawOrderAndDuplicates(t *testing.T) {
	initialRegions := []uint32{1001000}
	rawRegions := []uint32{9999999, 1001000, 9999999, 0, 42}
	_, _, loaded := writeRebuildFixture(t, PlatformPC, rebuildFixtureSlot{
		slot:         0,
		active:       true,
		version:      230,
		anchorAt:     accountIDTestFirstAnchorAt,
		regions:      initialRegions,
		menuSize:     16,
		tutorialSize: 32,
		worldSizes:   [5]uint32{16, 16, 32, 32, 64},
	})

	rebuilt, err := rebuildSlotWithRegions(loaded, 0, rawRegions)
	if err != nil {
		t.Fatalf("rebuildSlotWithRegions: %v", err)
	}

	countAt, _, _, _ := unlockedRegionsBounds(loaded, 0)
	preRegsLen := countAt - (eventFlagTestPCSlotDataBase)

	count := binary.LittleEndian.Uint32(rebuilt[preRegsLen:])
	if count != uint32(len(rawRegions)) {
		t.Fatalf("region count = %d, want %d", count, len(rawRegions))
	}
	for i, expected := range rawRegions {
		got := binary.LittleEndian.Uint32(rebuilt[preRegsLen+4+int64(i)*4:])
		if got != expected {
			t.Errorf("region[%d] = %d, want %d", i, got, expected)
		}
	}
}

func TestRebuildSlotRejectsCorruptedHeaderWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		fixture     rebuildFixtureSlot
		expectedErr string
	}{
		{
			name: "inactive slot",
			fixture: rebuildFixtureSlot{
				slot: 0, active: false, version: 230, anchorAt: accountIDTestFirstAnchorAt,
				regions: []uint32{1001000}, menuSize: 16, tutorialSize: 32,
			},
			expectedErr: "is not active",
		},
		{
			name: "no slot version",
			fixture: rebuildFixtureSlot{
				slot: 0, active: true, version: 0, breakHeader: "version_zero", anchorAt: accountIDTestFirstAnchorAt,
				regions: []uint32{1001000}, menuSize: 16, tutorialSize: 32,
			},
			expectedErr: "declares no slot version",
		},
		{
			name: "no anchor",
			fixture: rebuildFixtureSlot{
				slot: 0, active: true, version: 230, breakHeader: "no_anchor", anchorAt: accountIDTestFirstAnchorAt,
				regions: []uint32{1001000}, menuSize: 16, tutorialSize: 32,
			},
			expectedErr: "carries no gesture anchor",
		},
		{
			name: "broken menu profile header",
			fixture: rebuildFixtureSlot{
				slot: 0, active: true, version: 230, breakHeader: "broken_menu", anchorAt: accountIDTestFirstAnchorAt,
				regions: []uint32{1001000}, menuSize: 16, tutorialSize: 32,
			},
			expectedErr: "menu profile size",
		},
		{
			name: "broken tutorial data header",
			fixture: rebuildFixtureSlot{
				slot: 0, active: true, version: 230, breakHeader: "broken_tutorial", anchorAt: accountIDTestFirstAnchorAt,
				regions: []uint32{1001000}, menuSize: 16, tutorialSize: 32,
			},
			expectedErr: "tutorial size",
		},
		{
			name: "broken world block header",
			fixture: rebuildFixtureSlot{
				slot: 0, active: true, version: 230, breakHeader: "broken_world", anchorAt: accountIDTestFirstAnchorAt,
				regions: []uint32{1001000}, menuSize: 16, tutorialSize: 32,
			},
			expectedErr: "declares a world block 0 size of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, loaded := writeRebuildFixture(t, PlatformPC, tt.fixture)

			_, err := rebuildSlotWithRegions(loaded, 0, []uint32{1001000, 2002000})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestRebuildSlotDoesNotMutateSnapshotOrRevisionOrUndoOrOwnedRegistry(t *testing.T) {
	initialRegions := []uint32{1001000, 1001001}
	_, _, loaded := writeRebuildFixture(t, PlatformPC, rebuildFixtureSlot{
		slot:         0,
		active:       true,
		version:      230,
		anchorAt:     accountIDTestFirstAnchorAt,
		regions:      initialRegions,
		menuSize:     16,
		tutorialSize: 32,
		worldSizes:   [5]uint32{16, 16, 32, 32, 64},
	})

	snapshotBefore := append([]byte(nil), loaded.snapshot.data...)
	revBefore := loaded.session.revisionString()
	dirtyBefore := loaded.session.dirty
	undoBefore := loaded.session.undo
	ownedSeqBefore := loaded.session.ownedSeq
	ownedIDCountBefore := len(loaded.session.ownedByID)
	ownedLocatorCountBefore := len(loaded.session.ownedByLocator)

	newRegions := []uint32{1001000, 1001001, 6100000, 6200000}
	rebuilt, err := rebuildSlotWithRegions(loaded, 0, newRegions)
	if err != nil {
		t.Fatalf("rebuildSlotWithRegions: %v", err)
	}
	if len(rebuilt) != characterSlotDataSize {
		t.Fatalf("rebuilt size = %d", len(rebuilt))
	}

	// 1. Snapshot bytes are identical
	if !bytes.Equal(loaded.snapshot.data, snapshotBefore) {
		t.Errorf("snapshot data was mutated during rebuild")
	}

	// 2. Revision is unchanged
	if loaded.session.revisionString() != revBefore {
		t.Errorf("revision changed from %s to %s", revBefore, loaded.session.revisionString())
	}

	// 3. Dirty state is unchanged
	if loaded.session.dirty != dirtyBefore {
		t.Errorf("dirty state changed from %v to %v", dirtyBefore, loaded.session.dirty)
	}

	// 4. Undo point is unchanged
	if loaded.session.undo != undoBefore {
		t.Errorf("undo state changed from %v to %v", undoBefore, loaded.session.undo)
	}

	// 5. Owned item registry is unchanged
	if loaded.session.ownedSeq != ownedSeqBefore {
		t.Errorf("ownedSeq changed from %d to %d", ownedSeqBefore, loaded.session.ownedSeq)
	}
	if len(loaded.session.ownedByID) != ownedIDCountBefore {
		t.Errorf("ownedByID registry changed count from %d to %d", ownedIDCountBefore, len(loaded.session.ownedByID))
	}
	if len(loaded.session.ownedByLocator) != ownedLocatorCountBefore {
		t.Errorf("ownedByLocator registry changed count from %d to %d", ownedLocatorCountBefore, len(loaded.session.ownedByLocator))
	}
}
