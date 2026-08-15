package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Distances of the account-identifier chain, restated literally so a changed
// implementation constant fails here instead of moving both sides together.
const (
	accountIDTestEventFlagsSize   = 0x1BF99F + 1 // bitfield plus its terminator
	accountIDTestCoordinatesSize  = 61
	accountIDTestSpawnFixedSize   = 10
	accountIDTestNetManSize       = 0x20004
	accountIDTestTrailingOffset   = 12 + 12 + 16
	accountIDTestGlobalOffset     = 0x04
	accountIDTestSlotVersion      = 230 // both version-gated spawn fields present
	accountIDTestSpawnGatedExtra  = 4 + 1
	accountIDTestWorldBlockCount  = 5
	accountIDTestBrokenWorldBlock = 0x10000 // at the first world block ceiling, so it is rejected

	// The anchor is also the GaItem marker, and the GaItem table in front of it
	// starts at slot offset 0x20 and holds 5120 eight-byte records, so this is the
	// first position an anchor may occupy in a slot the write validation reloads.
	accountIDTestFirstAnchorAt = 0x20 + 5120*8
)

// accountIDTestSlot is the synthetic content of one PC slot. anchorAt and
// tutorialSize differ per slot so two active slots land on different dynamic
// account-identifier offsets.
type accountIDTestSlot struct {
	slot         int
	active       bool
	anchorAt     int64
	tutorialSize uint32
	worldSizes   [accountIDTestWorldBlockCount]uint32
	breakWorld   bool
}

// writeAccountIDFixture builds a synthetic PC container under t.TempDir() and
// reports its path together with the expected account-identifier offset of every
// described slot.
func writeAccountIDFixture(
	t *testing.T, platform Platform, slots []accountIDTestSlot,
) (string, map[int]int64) {
	t.Helper()

	var data []byte
	var userDataBase, slotBase, slotStride int64
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userDataBase, slotBase, slotStride =
			pcUserData10DataOffset, eventFlagTestPCSlotDataBase, eventFlagTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userDataBase, slotBase, slotStride =
			ps4UserData10DataOffset, eventFlagTestPS4SlotDataBase, eventFlagTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	offsets := make(map[int]int64, len(slots))
	for _, content := range slots {
		base := slotBase + int64(content.slot)*slotStride
		if content.active {
			data[userDataBase+userData10ActiveFlagsOffset+int64(content.slot)] = 1
		}
		binary.LittleEndian.PutUint32(data[base:], accountIDTestSlotVersion)
		copy(data[base+content.anchorAt:], eventFlagTestAnchor)

		put := func(at int64, value uint32) {
			binary.LittleEndian.PutUint32(data[base+at:], value)
		}
		// The chain in front of the bitfield: no projectile, no unlocked region and
		// an empty menu payload, then the declared tutorial payload.
		at := content.anchorAt + eventFlagTestMenuHeaderAt
		put(at+4, 0)
		at += eventFlagTestDynamicHeader + eventFlagTestTrophySize + eventFlagTestGaItemSize
		put(at+4, content.tutorialSize)
		at += eventFlagTestDynamicHeader + int64(content.tutorialSize) + eventFlagTestScalarsSize

		// The chain behind the bitfield: five size-prefixed world blocks, the real
		// 61-byte player coordinates, the version-gated spawn fields and NetMan.
		at += accountIDTestEventFlagsSize
		for index, size := range content.worldSizes {
			declared := size
			if content.breakWorld && index == 0 {
				declared = accountIDTestBrokenWorldBlock
			}
			put(at, declared)
			at += 4 + int64(size)
		}
		at += accountIDTestCoordinatesSize + accountIDTestSpawnFixedSize +
			accountIDTestSpawnGatedExtra + accountIDTestNetManSize + accountIDTestTrailingOffset
		offsets[content.slot] = base + at
	}

	path := filepath.Join(t.TempDir(), "account_id.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path, offsets
}

// accountIDTestSlots is the shared content: two active slots whose dynamic
// offsets differ, plus one residual slot that must never be touched.
func accountIDTestSlots() []accountIDTestSlot {
	return []accountIDTestSlot{
		{slot: 1, active: true, anchorAt: accountIDTestFirstAnchorAt, tutorialSize: 0x321,
			worldSizes: [accountIDTestWorldBlockCount]uint32{16, 32, 48, 64, 80}},
		{slot: 4, active: true, anchorAt: accountIDTestFirstAnchorAt + 0xE0, tutorialSize: 0x400,
			worldSizes: [accountIDTestWorldBlockCount]uint32{0, 8, 128, 0, 24}},
		{slot: 7, active: false, anchorAt: accountIDTestFirstAnchorAt, tutorialSize: 0x321,
			worldSizes: [accountIDTestWorldBlockCount]uint32{16, 32, 48, 64, 80}},
	}
}

func snapshotOf(t *testing.T, engine *Engine, saveSessionID string) []byte {
	t.Helper()

	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		t.Fatalf("session %q is not registered", saveSessionID)
	}
	return loaded.snapshot.data
}

func TestSetSaveAccountIDWritesTheGlobalAndEveryActiveSlotCopy(t *testing.T) {
	const accountID = "1311768467463790320"

	path, offsets := writeAccountIDFixture(t, PlatformPC, accountIDTestSlots())
	engine := New()
	loaded, err := engine.LoadSave(path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if offsets[1] == offsets[4] {
		t.Fatalf("both active slots resolve to offset 0x%X; the fixture is not dynamic", offsets[1])
	}
	residualBefore := append([]byte(nil),
		snapshotOf(t, engine, loaded.SaveSessionID)[eventFlagTestPCSlotDataBase+
			7*eventFlagTestPCSlotStride:][:eventFlagTestSlotDataSize]...)

	result, err := engine.SetSaveAccountID(loaded.SaveSessionID, accountID, "0")
	if err != nil {
		t.Fatalf("SetSaveAccountID: %v", err)
	}
	want := SetSaveAccountIDResult{SaveSessionID: loaded.SaveSessionID, SaveRevision: "1"}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	expected := uint64(0x123456789ABCDEF0)
	snapshot := snapshotOf(t, engine, loaded.SaveSessionID)
	globalAt := int64(pcUserData10DataOffset + accountIDTestGlobalOffset)
	if stored := binary.LittleEndian.Uint64(snapshot[globalAt:]); stored != expected {
		t.Errorf("global copy = %d, want the requested identifier", stored)
	}
	for _, slot := range []int{1, 4} {
		if stored := binary.LittleEndian.Uint64(snapshot[offsets[slot]:]); stored != expected {
			t.Errorf("slot %d copy at 0x%X = %d, want the requested identifier",
				slot, offsets[slot], stored)
		}
	}
	residualAfter := snapshot[eventFlagTestPCSlotDataBase+
		7*eventFlagTestPCSlotStride:][:eventFlagTestSlotDataSize]
	if !bytes.Equal(residualBefore, residualAfter) {
		t.Error("the inactive slot changed; it must stay byte for byte")
	}

	// WriteSave and a fresh LoadSave must preserve every copy.
	target := filepath.Join(t.TempDir(), "written.sl2")
	if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	reloadEngine := New()
	reloaded, err := reloadEngine.LoadSave(target, "pc")
	if err != nil {
		t.Fatalf("reload written target: %v", err)
	}
	written := snapshotOf(t, reloadEngine, reloaded.SaveSessionID)
	if stored := binary.LittleEndian.Uint64(written[globalAt:]); stored != expected {
		t.Errorf("reloaded global copy = %d, want the requested identifier", stored)
	}
	for _, slot := range []int{1, 4} {
		if stored := binary.LittleEndian.Uint64(written[offsets[slot]:]); stored != expected {
			t.Errorf("reloaded slot %d copy = %d, want the requested identifier", slot, stored)
		}
	}
}

func TestSetSaveAccountIDRejectsAnUnparseableActiveSlot(t *testing.T) {
	slots := accountIDTestSlots()
	slots[1].breakWorld = true

	path, _ := writeAccountIDFixture(t, PlatformPC, slots)
	engine := New()
	loaded, err := engine.LoadSave(path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := append([]byte(nil), snapshotOf(t, engine, loaded.SaveSessionID)...)

	if _, err := engine.SetSaveAccountID(loaded.SaveSessionID, "12345", "0"); err == nil {
		t.Fatal("SetSaveAccountID accepted a slot whose world block is unparseable")
	}
	if !bytes.Equal(before, snapshotOf(t, engine, loaded.SaveSessionID)) {
		t.Error("a rejected mutation changed the snapshot")
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Error("a rejected mutation marked the session dirty")
	}
	if revision := engine.sessions[loaded.SaveSessionID].session.revisionString(); revision != "0" {
		t.Errorf("revision = %q, want 0; a rejected mutation must not advance it", revision)
	}
}

func TestSetSaveAccountIDRejectsPS4WithoutMutation(t *testing.T) {
	path, _ := writeAccountIDFixture(t, PlatformPS4, accountIDTestSlots())
	engine := New()
	loaded, err := engine.LoadSave(path, "ps4")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := append([]byte(nil), snapshotOf(t, engine, loaded.SaveSessionID)...)

	_, err = engine.SetSaveAccountID(loaded.SaveSessionID, "12345", "0")
	if err == nil || !strings.Contains(err.Error(), "PC saves only") {
		t.Fatalf("PS4 error = %v, want an explicit PC-only rejection", err)
	}
	if !bytes.Equal(before, snapshotOf(t, engine, loaded.SaveSessionID)) {
		t.Error("a rejected PS4 mutation changed the snapshot")
	}
}

func TestSetSaveAccountIDRejectsANonCanonicalIdentifier(t *testing.T) {
	path, _ := writeAccountIDFixture(t, PlatformPC, accountIDTestSlots())
	engine := New()
	loaded, err := engine.LoadSave(path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := append([]byte(nil), snapshotOf(t, engine, loaded.SaveSessionID)...)

	for _, accountID := range []string{
		"", "007", "-1", "+1", " 1", "1 ", "0x1", "1e3",
		"18446744073709551616", // one past the uint64 maximum
	} {
		result, err := engine.SetSaveAccountID(loaded.SaveSessionID, accountID, "0")
		if err == nil {
			t.Errorf("accountID %q was accepted", accountID)
			continue
		}
		if result != (SetSaveAccountIDResult{}) {
			t.Errorf("accountID %q returned %+v, want the zero value", accountID, result)
		}
		if accountID != "" && strings.Contains(err.Error(), accountID) {
			t.Errorf("the error for %q repeats the rejected identifier", accountID)
		}
	}
	if !bytes.Equal(before, snapshotOf(t, engine, loaded.SaveSessionID)) {
		t.Error("a rejected identifier changed the snapshot")
	}
}
