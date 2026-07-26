package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests exercise the issue-#10 fix: a PC save's global UserData10 SteamID
// and every active slot's own SteamID (a u64 inside TrailingFixedBlock at a
// per-slot dynamic offset) must stay in sync, or the game rejects the save as
// corrupt.
//
// Fixtures under tmp/save/ are gitignored and treated strictly read-only: every
// test LoadSaves the original (a read) and writes only into t.TempDir(). The
// regression test copies the fixture into t.TempDir() before touching it.

// pcMultiSlotFixturePath returns a real PC save with several active slots, or
// skips loudly when no such fixture is checked in locally.
func pcMultiSlotFixturePath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		"../../tmp/save/ER0000-out.sl2",
		"../../tmp/save/cos-nowego.dat",
		"../../tmp/save/cos-nowego-2.dat",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("SKIP(real fixture): no multi-slot PC save present under tmp/save/ (gitignored) — synthetic-only run, not full game-parity proof")
	return ""
}

// ps4FixturePath returns a real PS4 save with active slots, or skips loudly.
func ps4FixturePath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		"../../tmp/save/oisisk_ps4.dat",
		"../../tmp/save/OiSiSiSBack-3char-1.2.0.dat",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("SKIP(real fixture): no PS4 save present under tmp/save/ (gitignored)")
	return ""
}

func readSlotSteamIDField(t *testing.T, slot *SaveSlot) uint64 {
	t.Helper()
	off, err := slotSteamIDFieldOffset(slot)
	if err != nil {
		t.Fatalf("slotSteamIDFieldOffset: %v", err)
	}
	return binary.LittleEndian.Uint64(slot.Data[off : off+8])
}

// assertBytesDifferOnlyIn fails unless a and b are byte-identical everywhere
// except the half-open range [start, end).
func assertBytesDifferOnlyIn(t *testing.T, a, b []byte, start, end int, label string) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("%s: length %d != %d", label, len(a), len(b))
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		if i >= start && i < end {
			continue
		}
		t.Fatalf("%s: unexpected change at 0x%X (0x%02X -> 0x%02X); only [0x%X,0x%X) may differ",
			label, i, a[i], b[i], start, end)
	}
}

func copyToTemp(t *testing.T, src string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("write temp copy: %v", err)
	}
	return dst
}

func activeSlotIndices(save *SaveFile) []int {
	var idx []int
	for i := 0; i < 10; i++ {
		if save.ActiveSlots[i] && save.Slots[i].Version != 0 && save.Slots[i].UnlockedRegionsOffset != 0 {
			idx = append(idx, i)
		}
	}
	return idx
}

// TestSlotSteamIDSyncPCUpdatesGlobalAndEverySlot covers the core PC A→B flow:
// after changing the global SteamID and saving, UserData10 AND every active slot
// carry B; inactive slots are byte-exact; every non-SteamID byte of each active
// slot (hence all other TrailingFixedBlock fields) is byte-exact; and the write
// survives SaveFile + MD5 + a fresh LoadSave. Also proves the per-slot offset is
// genuinely dynamic (distinct across slots).
func TestSlotSteamIDSyncPCUpdatesGlobalAndEverySlot(t *testing.T) {
	path := pcMultiSlotFixturePath(t)
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if save.Platform != PlatformPC {
		t.Fatalf("expected PC fixture, got %s", save.Platform)
	}
	active := activeSlotIndices(save)
	if len(active) < 2 {
		t.Fatalf("need >= 2 active slots for dynamic-offset coverage, got %d", len(active))
	}

	// Prove offsets differ between slots (dynamic, not fixed).
	offsets := map[int]int{}
	distinct := map[int]struct{}{}
	for _, i := range active {
		off, err := slotSteamIDFieldOffset(&save.Slots[i])
		if err != nil {
			t.Fatalf("slot %d offset: %v", i, err)
		}
		offsets[i] = off
		distinct[off] = struct{}{}
	}
	if len(distinct) < 2 {
		t.Fatalf("expected distinct per-slot SteamID offsets, all equal: %v", offsets)
	}

	// Snapshots for byte-exactness assertions.
	activeBefore := map[int][]byte{}
	for _, i := range active {
		activeBefore[i] = append([]byte(nil), save.Slots[i].Data...)
	}
	inactiveBefore := map[int][]byte{}
	for i := 0; i < 10; i++ {
		if !save.ActiveSlots[i] {
			inactiveBefore[i] = append([]byte(nil), save.Slots[i].Data...)
		}
	}
	prefixBefore := append([]byte(nil), save.UserData10.Data[0:4]...)

	globalBefore := save.SteamID
	globalAfter := globalBefore + 1
	if globalAfter == 0 {
		t.Fatal("fixture SteamID cannot be incremented")
	}
	save.SteamID = globalAfter

	outPath := filepath.Join(t.TempDir(), "out.sl2")
	if err := save.SaveFile(outPath); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	// In-memory: active slots changed only in their 8 SteamID bytes.
	for _, i := range active {
		off := offsets[i]
		assertBytesDifferOnlyIn(t, activeBefore[i], save.Slots[i].Data, off, off+8,
			"active slot in-memory")
	}
	// In-memory: inactive slots untouched.
	for i, before := range inactiveBefore {
		if !bytes.Equal(before, save.Slots[i].Data) {
			t.Fatalf("inactive slot %d mutated in memory", i)
		}
	}

	// Reload from disk and re-check both representations.
	reloaded, err := LoadSave(outPath)
	if err != nil {
		t.Fatalf("LoadSave (reload): %v", err)
	}
	if reloaded.SteamID != globalAfter {
		t.Fatalf("reloaded global SteamID = %d, want %d", reloaded.SteamID, globalAfter)
	}
	got := binary.LittleEndian.Uint64(reloaded.UserData10.Data[SteamIDOffset : SteamIDOffset+8])
	if got != globalAfter {
		t.Errorf("reloaded UserData10 SteamID = %d, want %d", got, globalAfter)
	}
	if gotPrefix := reloaded.UserData10.Data[0:4]; !bytes.Equal(gotPrefix, prefixBefore) {
		t.Errorf("UserData10 metadata prefix changed: %x -> %x", prefixBefore, gotPrefix)
	}
	for _, i := range active {
		if id := readSlotSteamIDField(t, &reloaded.Slots[i]); id != globalAfter {
			t.Errorf("reloaded slot %d SteamID = %d, want %d", i, id, globalAfter)
		}
		off := offsets[i]
		assertBytesDifferOnlyIn(t, activeBefore[i], reloaded.Slots[i].Data, off, off+8,
			"active slot reloaded")
	}
	for i, before := range inactiveBefore {
		if !bytes.Equal(before, reloaded.Slots[i].Data) {
			t.Errorf("inactive slot %d not byte-exact after reload", i)
		}
	}
	if err := reloaded.ValidateSteamIDConsistency(); err != nil {
		t.Errorf("reloaded save failed consistency check: %v", err)
	}
}

// TestValidateSteamIDConsistencyDetectsMismatch reproduces the issue-#10 corrupt
// state in memory (global changed, per-slot copies left stale) and asserts the
// SaveFile-level validator catches it — while a pristine fixture validates clean.
func TestValidateSteamIDConsistencyDetectsMismatch(t *testing.T) {
	path := pcMultiSlotFixturePath(t)
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if err := save.ValidateSteamIDConsistency(); err != nil {
		t.Fatalf("pristine fixture should validate clean, got: %v", err)
	}
	// Change ONLY the global id — exactly the pre-fix bug.
	save.SteamID++
	if err := save.ValidateSteamIDConsistency(); err == nil {
		t.Fatal("expected mismatch error after changing only global SteamID, got nil")
	}
}

// TestSyncActiveSlotSteamIDsFailsClosed asserts that a preflight failure on any
// active slot aborts before a single slot is mutated (atomic, fail-closed).
func TestSyncActiveSlotSteamIDsFailsClosed(t *testing.T) {
	path := pcMultiSlotFixturePath(t)
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	active := activeSlotIndices(save)
	if len(active) < 2 {
		t.Fatalf("need >= 2 active slots, got %d", len(active))
	}

	before := map[int]uint64{}
	for _, i := range active {
		before[i] = readSlotSteamIDField(t, &save.Slots[i])
	}

	// Damage the LAST active slot so its offset is unresolvable; the earlier
	// slots would otherwise be written first.
	broken := active[len(active)-1]
	save.Slots[broken].SectionMap = nil

	save.SteamID += 1000
	if err := save.syncActiveSlotSteamIDs(); err == nil {
		t.Fatal("expected fail-closed error, got nil")
	}

	// No active slot's SteamID may have changed — not even the slots that would
	// have been written before the broken one.
	for _, i := range active {
		if i == broken {
			continue // its parse is broken; skip re-read
		}
		if got := readSlotSteamIDField(t, &save.Slots[i]); got != before[i] {
			t.Errorf("slot %d SteamID mutated to %d despite fail-closed abort (was %d)", i, got, before[i])
		}
	}
}

// TestSaveFileFailedSyncLeavesNoPartialMutation drives the full public SaveFile
// path (not just syncActiveSlotSteamIDs) and asserts that when the slot SteamID
// sync fails, SaveFile leaves NO partial in-memory mutation. This specifically
// catches the pre-fix ordering bug where flushMetadata ran before the sync: a
// failed sync then returned an error but had already re-serialized the new
// SteamID/metadata into UserData10, contradicting the fail-closed guarantee.
//
// The last active slot's SectionMap is nulled so ValidateSlotIntegrity (which
// never reads SectionMap) still passes, but slotSteamIDFieldOffset fails during
// sync — reproducing a mid-write failure through the real entrypoint.
func TestSaveFileFailedSyncLeavesNoPartialMutation(t *testing.T) {
	path := pcMultiSlotFixturePath(t)
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if save.Platform != PlatformPC {
		t.Fatalf("expected PC fixture, got %s", save.Platform)
	}
	active := activeSlotIndices(save)
	if len(active) < 2 {
		t.Fatalf("need >= 2 active slots, got %d", len(active))
	}

	// Set a new global id; if flushMetadata runs prematurely it will burn this
	// into UserData10 before the sync failure aborts the write.
	save.SteamID += 1000

	// Byte-exact snapshots of everything SaveFile could mutate.
	ud10Before := append([]byte(nil), save.UserData10.Data...)
	slotDataBefore := map[int][]byte{}
	slotIDBefore := map[int]uint64{}
	for i := 0; i < 10; i++ {
		slotDataBefore[i] = append([]byte(nil), save.Slots[i].Data...)
		slotIDBefore[i] = save.Slots[i].SteamID
	}

	// Break ONLY the last active slot's SectionMap: ValidateSlotIntegrity still
	// passes, but the sync's offset resolution fails on it.
	broken := active[len(active)-1]
	save.Slots[broken].SectionMap = nil

	outPath := filepath.Join(t.TempDir(), "should-not-exist.sl2")
	err = save.SaveFile(outPath)
	if err == nil {
		t.Fatal("expected SaveFile to fail on unresolvable slot SteamID, got nil")
	}
	if !strings.Contains(err.Error(), "slot SteamID sync failed") {
		t.Fatalf("expected 'slot SteamID sync failed' error, got: %v", err)
	}

	// The output file must not exist.
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		t.Fatalf("output file exists after failed SaveFile (stat err: %v)", statErr)
	}

	// UserData10 must be byte-exact: flushMetadata must not have run.
	if !bytes.Equal(ud10Before, save.UserData10.Data) {
		t.Error("UserData10.Data mutated despite failed SaveFile (premature flushMetadata)")
	}

	// Every slot's Data and SteamID must be byte-exact: no slot may be mutated,
	// not even active slots preceding the broken one, nor inactive slots.
	for i := 0; i < 10; i++ {
		if !bytes.Equal(slotDataBefore[i], save.Slots[i].Data) {
			t.Errorf("slot %d Data mutated despite failed SaveFile", i)
		}
		if save.Slots[i].SteamID != slotIDBefore[i] {
			t.Errorf("slot %d SaveSlot.SteamID mutated to %d (was %d) despite failed SaveFile",
				i, save.Slots[i].SteamID, slotIDBefore[i])
		}
	}
}

// TestPS4ToPCConversionWritesBothRepresentations mirrors ExecuteConversion's PC
// branch: the target Steam id must land in both UserData10 and every active
// slot, even though the source PS4 slots carried a different (zero) id.
func TestPS4ToPCConversionWritesBothRepresentations(t *testing.T) {
	path := ps4FixturePath(t)
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if save.Platform != PlatformPS {
		t.Fatalf("expected PS4 fixture, got %s", save.Platform)
	}
	active := activeSlotIndices(save)
	if len(active) == 0 {
		t.Fatal("no active slots in PS4 fixture")
	}
	// Source slot ids are the PS4 zero identity — deliberately != target.
	for _, i := range active {
		if id := readSlotSteamIDField(t, &save.Slots[i]); id == 0 {
			continue
		} else {
			t.Logf("note: PS4 slot %d already carries non-zero id %d", i, id)
		}
	}

	const targetSteamID = uint64(76561198000000042)
	// Emulate ExecuteConversion(target="PC"); keep Encrypted=false so LoadSave
	// (which accepts only raw BND4) can reload the result in-test.
	save.Platform = PlatformPC
	save.Encrypted = false
	save.SteamID = targetSteamID

	outPath := filepath.Join(t.TempDir(), "converted.sl2")
	if err := save.SaveFile(outPath); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	reloaded, err := LoadSave(outPath)
	if err != nil {
		t.Fatalf("LoadSave (reload): %v", err)
	}
	if reloaded.Platform != PlatformPC {
		t.Fatalf("reloaded platform = %s, want PC", reloaded.Platform)
	}
	if reloaded.SteamID != targetSteamID {
		t.Errorf("reloaded global SteamID = %d, want %d", reloaded.SteamID, targetSteamID)
	}
	for _, i := range active {
		if id := readSlotSteamIDField(t, &reloaded.Slots[i]); id != targetSteamID {
			t.Errorf("reloaded slot %d SteamID = %d, want %d", i, id, targetSteamID)
		}
	}
	if err := reloaded.ValidateSteamIDConsistency(); err != nil {
		t.Errorf("converted save failed consistency check: %v", err)
	}
}

// TestPS4SaveWithoutConversionUntouched asserts that saving a PS4 file never
// writes a PC-only per-slot SteamID mutation, even if SaveFile.SteamID is set.
func TestPS4SaveWithoutConversionUntouched(t *testing.T) {
	path := ps4FixturePath(t)
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	active := activeSlotIndices(save)
	if len(active) == 0 {
		t.Fatal("no active slots in PS4 fixture")
	}
	slotBefore := map[int][]byte{}
	for _, i := range active {
		slotBefore[i] = append([]byte(nil), save.Slots[i].Data...)
	}

	// A stray global value must not leak into PS4 slots.
	save.SteamID = 76561198000000999

	outPath := filepath.Join(t.TempDir(), "ps4-out.dat")
	if err := save.SaveFile(outPath); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	reloaded, err := LoadSave(outPath)
	if err != nil {
		t.Fatalf("LoadSave (reload): %v", err)
	}
	if reloaded.Platform != PlatformPS {
		t.Fatalf("reloaded platform = %s, want PS4", reloaded.Platform)
	}
	for _, i := range active {
		if !bytes.Equal(slotBefore[i], reloaded.Slots[i].Data) {
			off, _ := slotSteamIDFieldOffset(&reloaded.Slots[i])
			t.Errorf("PS4 slot %d changed on plain save (slot SteamID offset 0x%X)", i, off)
		}
	}
}

// TestIssue10RegressionOnFixtureCopy reproduces the issue-#10 user flow entirely
// on a COPY of a fixture (never the original .sl2): open, change the account's
// Steam id, save back onto the copy, reload. Before the fix the reloaded save
// held a global/slot mismatch (the game's "corrupt save"); now both
// representations agree and the consistency check passes.
func TestIssue10RegressionOnFixtureCopy(t *testing.T) {
	src := pcMultiSlotFixturePath(t)
	copyPath := copyToTemp(t, src)

	save, err := LoadSave(copyPath)
	if err != nil {
		t.Fatalf("LoadSave(copy): %v", err)
	}
	active := activeSlotIndices(save)
	if len(active) == 0 {
		t.Fatal("no active slots")
	}

	newID := save.SteamID + 7
	save.SteamID = newID

	// Save back onto the copy (the real user action), then reopen the copy.
	if err := save.SaveFile(copyPath); err != nil {
		t.Fatalf("SaveFile(copy): %v", err)
	}
	reloaded, err := LoadSave(copyPath)
	if err != nil {
		t.Fatalf("LoadSave(copy reload): %v", err)
	}
	if err := reloaded.ValidateSteamIDConsistency(); err != nil {
		t.Fatalf("issue #10 NOT fixed — reloaded save is inconsistent: %v", err)
	}
	for _, i := range active {
		if id := readSlotSteamIDField(t, &reloaded.Slots[i]); id != newID {
			t.Errorf("slot %d SteamID = %d, want %d", i, id, newID)
		}
	}
	if reloaded.SteamID != newID {
		t.Errorf("global SteamID = %d, want %d", reloaded.SteamID, newID)
	}
}
