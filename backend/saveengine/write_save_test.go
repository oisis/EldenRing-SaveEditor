package saveengine

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func serializationTestLoaded(platform Platform, data []byte) *loadedSave {
	return &loadedSave{
		session:  &Session{platform: platform},
		snapshot: &codec{data: data},
	}
}

func TestSerializeContainerRefreshesOnlyPCChecksums(t *testing.T) {
	// The bytes after the minimum accepted PC container stand in for UserData11,
	// which has no checksum prefix in the 1.5.8/1.6.8 writer.
	original := make([]byte, pcFixtureSize+37)
	for index := range original {
		original[index] = byte(index*31 + 7)
	}
	for slot := 0; slot < pcSlotCount; slot++ {
		checksumAt := pcSlotsOffset + int64(slot)*pcSlotBlockSize
		clear(original[checksumAt : checksumAt+md5.Size])
	}
	clear(original[pcUserData10Offset : pcUserData10Offset+md5.Size])
	before := bytes.Clone(original)

	candidate, err := serializeContainer(serializationTestLoaded(PlatformPC, original))
	if err != nil {
		t.Fatalf("serializeContainer: %v", err)
	}

	unchangedFrom := int64(0)
	var zeroChecksum [md5.Size]byte
	for slot := 0; slot < pcSlotCount; slot++ {
		checksumAt := pcSlotsOffset + int64(slot)*pcSlotBlockSize
		dataAt := checksumAt + md5.Size
		if !bytes.Equal(candidate[unchangedFrom:checksumAt], before[unchangedFrom:checksumAt]) {
			t.Errorf("serialized PC candidate changed bytes before the checksum of slot %d", slot)
		}
		checksum := md5.Sum(before[dataAt : dataAt+pcSlotBlockSize-md5.Size])
		if !bytes.Equal(candidate[checksumAt:dataAt], checksum[:]) {
			t.Errorf("slot %d checksum = % X, want % X", slot, candidate[checksumAt:dataAt], checksum)
		}
		if bytes.Equal(candidate[checksumAt:dataAt], zeroChecksum[:]) {
			t.Errorf("slot %d retained an all-zero checksum", slot)
		}
		unchangedFrom = dataAt
	}
	if !bytes.Equal(
		candidate[unchangedFrom:pcUserData10Offset],
		before[unchangedFrom:pcUserData10Offset],
	) {
		t.Error("serialized PC candidate changed slot data before UserData10")
	}
	userDataAt := pcUserData10Offset + md5.Size
	userDataChecksum := md5.Sum(
		before[userDataAt : userDataAt+pcUserData10BlockSize-md5.Size],
	)
	if !bytes.Equal(candidate[pcUserData10Offset:userDataAt], userDataChecksum[:]) {
		t.Errorf("UserData10 checksum = % X, want % X",
			candidate[pcUserData10Offset:userDataAt], userDataChecksum)
	}
	if !bytes.Equal(candidate[userDataAt:], before[userDataAt:]) {
		t.Error("serialized PC candidate changed UserData10 or UserData11 data")
	}
	if !bytes.Equal(original, before) {
		t.Error("serializing a PC container changed the session snapshot")
	}
	if !bytes.Equal(candidate[pcFixtureSize:], before[pcFixtureSize:]) {
		t.Error("serializing a PC container changed the UserData11 tail")
	}
}

func TestSerializeContainerCopiesPS4WithoutTransformation(t *testing.T) {
	original := []byte{0xCB, 0x01, 0x9C, 0x2C, 0xAA, 0xBB}
	candidate, err := serializeContainer(serializationTestLoaded(PlatformPS4, original))
	if err != nil {
		t.Fatalf("serializeContainer: %v", err)
	}
	if !bytes.Equal(candidate, original) {
		t.Fatalf("serialized PS4 candidate = % X, want % X", candidate, original)
	}

	candidate[0] ^= 0xFF
	if original[0] != 0xCB {
		t.Error("serialized PS4 candidate aliases the session snapshot")
	}
}

func TestSerializeContainerRejectsUnknownPlatform(t *testing.T) {
	result, err := serializeContainer(serializationTestLoaded(Platform("other"), []byte{1, 2, 3}))
	if err == nil || !strings.Contains(err.Error(), `unknown save platform "other"`) {
		t.Fatalf("serializeContainer error = %v, want unknown-platform error", err)
	}
	if result != nil {
		t.Errorf("serializeContainer result = % X, want nil", result)
	}
}

func TestSerializeContainerRejectsShortPCSnapshotWithoutChangingIt(t *testing.T) {
	original := bytes.Repeat([]byte{0x5A}, pcHeaderSize)
	before := bytes.Clone(original)
	result, err := serializeContainer(serializationTestLoaded(PlatformPC, original))
	if err == nil || !strings.Contains(err.Error(), "cannot checksum PC slot 0") {
		t.Fatalf("serializeContainer error = %v, want short-slot error", err)
	}
	if result != nil {
		t.Errorf("serializeContainer result has %d bytes, want nil", len(result))
	}
	if !bytes.Equal(original, before) {
		t.Error("rejected PC serialization changed the session snapshot")
	}
}

func TestValidateSerializedReloadsOwnedItemsOnBothPlatformsWithoutTouchingTheSession(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			info, err := engine.LoadSave(writeOwnedItemContainerFixture(t, platform), "")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			engine.mutex.Lock()
			loaded := engine.sessions[info.SaveSessionID]
			candidate, err := serializeContainer(loaded)
			beforeRevision := loaded.session.revision
			beforeDirty := loaded.session.dirty
			beforeIDs := len(loaded.session.ownedByID)
			engine.mutex.Unlock()
			if err != nil {
				t.Fatalf("serializeContainer: %v", err)
			}
			beforeCandidate := bytes.Clone(candidate)

			if err := validateSerialized(candidate, platform); err != nil {
				t.Fatalf("validateSerialized: %v", err)
			}
			if !bytes.Equal(candidate, beforeCandidate) {
				t.Error("validation changed the serialized candidate")
			}

			engine.mutex.Lock()
			if loaded.session.revision != beforeRevision || loaded.session.dirty != beforeDirty ||
				len(loaded.session.ownedByID) != beforeIDs || len(loaded.session.ownedByLocator) != beforeIDs {
				t.Errorf(
					"validation changed the live session: revision %d, dirty %v, IDs %d/%d; want %d, %v, %d/%d",
					loaded.session.revision,
					loaded.session.dirty,
					len(loaded.session.ownedByID),
					len(loaded.session.ownedByLocator),
					beforeRevision,
					beforeDirty,
					beforeIDs,
					beforeIDs,
				)
			}
			engine.mutex.Unlock()
		})
	}
}

func TestValidateSerializedRejectsWrongPlatformAndMalformedContainer(t *testing.T) {
	data := make([]byte, pcFixtureSize)
	copy(data, pcHeader())
	candidate, err := serializeContainer(serializationTestLoaded(PlatformPC, data))
	if err != nil {
		t.Fatalf("serializeContainer: %v", err)
	}

	if err := validateSerialized(candidate, PlatformPS4); err == nil ||
		!strings.Contains(err.Error(), "is a pc save, expected ps4") {
		t.Errorf("wrong-platform error = %v", err)
	}
	if err := validateSerialized(candidate, Platform("other")); err == nil ||
		!strings.Contains(err.Error(), `unknown expected platform "other"`) {
		t.Errorf("unknown-platform error = %v", err)
	}

	malformed := bytes.Clone(candidate)
	binary.LittleEndian.PutUint32(malformed[pcEntryCountOffset:], pcEntryCount-1)
	if err := validateSerialized(malformed, PlatformPC); err == nil ||
		!strings.Contains(err.Error(), "declares 11 BND4 entries, want 12") {
		t.Errorf("malformed-container error = %v", err)
	}
	if err := validateSerialized([]byte{1, 2, 3}, PlatformPC); err == nil ||
		!strings.Contains(err.Error(), "too short to identify") {
		t.Errorf("short-candidate error = %v", err)
	}
	if err := validateSerialized([]byte{1, 2, 3, 4}, PlatformPC); err == nil ||
		!strings.Contains(err.Error(), "neither a native PC nor a native PS4 container") {
		t.Errorf("unknown-container error = %v", err)
	}
}

func TestValidateSerializedRejectsMalformedActiveOwnedItems(t *testing.T) {
	cases := map[string]struct {
		mutate  func([]byte)
		message string
	}{
		"missing GaItem marker": {
			mutate: func(data []byte) {
				slotBase := pcSlotDataOffset + ownedContainerTestSlot*pcSlotBlockSize
				clear(data[slotBase+ownedContainerTestAnchorAt : slotBase+ownedContainerTestAnchorAt+int64(len(gaItemAnchor))])
			},
			message: "slot carries no GaItem marker",
		},
		"unresolvable Inventory handle": {
			mutate: func(data []byte) {
				slotBase := pcSlotDataOffset + ownedContainerTestSlot*pcSlotBlockSize
				recordAt := slotBase + ownedContainerTestAnchorAt + inventoryHeldCommonOffset +
					ownedContainerTestCommonIndex*inventoryHeldRecordSize
				binary.LittleEndian.PutUint32(data[recordAt:], gaItemWeaponHandle|1)
			},
			message: "inventory record common/1: GaItem handle 0x80000001 has no record",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeOwnedItemContainerFixture(t, PlatformPC)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read synthetic fixture: %v", err)
			}
			testCase.mutate(data)
			candidate, err := serializeContainer(serializationTestLoaded(PlatformPC, data))
			if err != nil {
				t.Fatalf("serializeContainer: %v", err)
			}

			if err := validateSerialized(candidate, PlatformPC); err == nil ||
				!strings.Contains(err.Error(), testCase.message) {
				t.Errorf("validateSerialized error = %v, want it to contain %q", err, testCase.message)
			}
		})
	}
}

func TestValidateSerializedDoesNotInspectResidualSlotData(t *testing.T) {
	path := writeOwnedItemContainerFixture(t, PlatformPC)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read synthetic fixture: %v", err)
	}
	data[pcUserData10DataOffset+userData10ActiveFlagsOffset+ownedContainerTestSlot] = 0
	slotBase := pcSlotDataOffset + ownedContainerTestSlot*pcSlotBlockSize
	clear(data[slotBase+ownedContainerTestAnchorAt : slotBase+ownedContainerTestAnchorAt+int64(len(gaItemAnchor))])

	candidate, err := serializeContainer(serializationTestLoaded(PlatformPC, data))
	if err != nil {
		t.Fatalf("serializeContainer: %v", err)
	}
	if err := validateSerialized(candidate, PlatformPC); err != nil {
		t.Fatalf("validateSerialized inspected residual slot data: %v", err)
	}
}

func TestWriteAtomicallyCreatesAndReplacesRegularFile(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "save.sl2")

	if err := writeAtomically(target, []byte("first candidate")); err != nil {
		t.Fatalf("writeAtomically new target: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read new target: %v", err)
	}
	if string(data) != "first candidate" {
		t.Fatalf("new target = %q, want %q", data, "first candidate")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("stat new target: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o644 {
			t.Errorf("new target permissions = %04o, want 0644", got)
		}
	}

	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatalf("chmod existing target: %v", err)
	}
	if err := writeAtomically(target, []byte("replacement")); err != nil {
		t.Fatalf("writeAtomically existing target: %v", err)
	}
	data, err = os.ReadFile(target)
	if err != nil {
		t.Fatalf("read replaced target: %v", err)
	}
	if string(data) != "replacement" {
		t.Fatalf("replaced target = %q, want %q", data, "replacement")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("stat replaced target: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("replaced target permissions = %04o, want 0600", got)
		}
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read target directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".saveforge-write-") {
			t.Errorf("temporary file leaked after successful write: %q", entry.Name())
		}
	}
}

func TestWriteAtomicallyRejectsUnsafeTargetsWithoutChangingThem(t *testing.T) {
	directory := t.TempDir()
	if err := writeAtomically("", []byte("candidate")); err == nil ||
		!strings.Contains(err.Error(), "write target is empty") {
		t.Errorf("empty-target error = %v", err)
	}

	if err := writeAtomically(directory, []byte("candidate")); err == nil ||
		!strings.Contains(err.Error(), "is not a regular file") {
		t.Errorf("directory-target error = %v", err)
	}

	realTarget := filepath.Join(directory, "existing.sl2")
	if err := os.WriteFile(realTarget, []byte("unchanged"), 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	symlinkTarget := filepath.Join(directory, "save-link.sl2")
	if err := os.Symlink(realTarget, symlinkTarget); err != nil {
		t.Skipf("cannot create symlink on this platform: %v", err)
	}
	if err := writeAtomically(symlinkTarget, []byte("candidate")); err == nil ||
		!strings.Contains(err.Error(), "is not a regular file") {
		t.Errorf("symlink-target error = %v", err)
	}
	data, err := os.ReadFile(realTarget)
	if err != nil {
		t.Fatalf("read symlink destination: %v", err)
	}
	if string(data) != "unchanged" {
		t.Errorf("rejected symlink write changed destination to %q", data)
	}
}

func TestWriteAtomicallyRejectsMissingParentDirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing", "save.sl2")
	if err := writeAtomically(target, []byte("candidate")); err == nil ||
		!strings.Contains(err.Error(), "cannot create temporary save") {
		t.Fatalf("missing-parent error = %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("rejected write created target: %v", err)
	}
}
