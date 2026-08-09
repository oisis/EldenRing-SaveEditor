package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// pcFixtureSize and ps4FixtureSize are the smallest sizes the first stage
// accepts for each container.
const (
	pcFixtureSize  = int64(pcHeaderSize) + pcSlotsSize + pcUserData10BlockSize
	ps4FixtureSize = int64(ps4HeaderSize) + ps4SlotsSize + ps4UserData10Size
)

// writeFixture writes header at the start of a synthetic save file and grows it
// to size. The body stays sparse: the stage validates bounds, not content.
func writeFixture(t *testing.T, name string, header []byte, size int64) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(header); err != nil {
		t.Fatalf("write fixture header: %v", err)
	}
	if err := file.Truncate(size); err != nil {
		t.Fatalf("size fixture: %v", err)
	}
	return path
}

func pcHeader() []byte {
	header := make([]byte, pcHeaderSize)
	copy(header, pcMagic)
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount)
	return header
}

func ps4Header() []byte {
	header := make([]byte, ps4HeaderSize)
	copy(header, ps4Magic)
	for entry := 0; entry < ps4EntryCount; entry++ {
		offset := ps4EntryTableOffset + entry*ps4EntryStride
		binary.LittleEndian.PutUint32(header[offset:], uint32(ps4FirstEntryIndex+entry))
		binary.LittleEndian.PutUint32(header[offset+4:], ps4EntryMarker)
	}
	return header
}

func TestLoadSaveCreatesSessionForPCContainer(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)

	info, err := New().LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if info.SaveSessionID == "" {
		t.Error("saveSessionID is empty")
	}
	if info.Platform != string(PlatformPC) {
		t.Errorf("platform = %q, want %q", info.Platform, PlatformPC)
	}
	if info.Format != pcContainerFormat {
		t.Errorf("format = %q, want %q", info.Format, pcContainerFormat)
	}
	if info.UnsavedChanges {
		t.Error("a freshly loaded session reports unsaved changes")
	}
}

func TestLoadSaveCreatesSessionForPS4Container(t *testing.T) {
	path := writeFixture(t, "ps4.sl2", ps4Header(), ps4FixtureSize)

	info, err := New().LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if info.SaveSessionID == "" {
		t.Error("saveSessionID is empty")
	}
	if info.Platform != string(PlatformPS4) {
		t.Errorf("platform = %q, want %q", info.Platform, PlatformPS4)
	}
	if info.Format != ps4ContainerFormat {
		t.Errorf("format = %q, want %q", info.Format, ps4ContainerFormat)
	}
	if info.UnsavedChanges {
		t.Error("a freshly loaded session reports unsaved changes")
	}
}

func TestLoadSaveGivesEverySessionItsOwnIdentifier(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	first, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("first LoadSave: %v", err)
	}
	second, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("second LoadSave: %v", err)
	}
	if first.SaveSessionID == second.SaveSessionID {
		t.Errorf("two sessions share the identifier %q", first.SaveSessionID)
	}
	if len(engine.sessions) != 2 {
		t.Errorf("engine holds %d sessions, want 2", len(engine.sessions))
	}
}

func TestLoadSaveKeepsPrivateSnapshotOfTheSource(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// The source changes after a successful load. The session keeps the data it
	// was created from.
	if err := os.WriteFile(path, []byte("NOPE not a save"), 0o600); err != nil {
		t.Fatalf("overwrite source: %v", err)
	}

	loaded := engine.sessions[info.SaveSessionID]
	if loaded == nil {
		t.Fatalf("engine holds no session for %q", info.SaveSessionID)
	}
	if !bytes.HasPrefix(loaded.snapshot.data, pcMagic) {
		t.Errorf("snapshot starts with %q, want the original magic %q",
			loaded.snapshot.data[:min(len(loaded.snapshot.data), len(pcMagic))], pcMagic)
	}
	if loaded.snapshot.length() != pcFixtureSize {
		t.Errorf("snapshot length = 0x%X, want 0x%X", loaded.snapshot.length(), pcFixtureSize)
	}

	// The public metadata still carries no source data: no path, no bytes, and
	// no field beyond the four documented ones.
	if fields := reflect.TypeOf(info).NumField(); fields != 4 {
		t.Errorf("SessionInfo has %d fields, want the 4 metadata fields", fields)
	}
	for name, value := range map[string]string{
		"saveSessionID": info.SaveSessionID,
		"platform":      info.Platform,
		"format":        info.Format,
	} {
		if strings.Contains(value, path) || strings.Contains(value, string(pcMagic)) {
			t.Errorf("%s = %q exposes the source path or its content", name, value)
		}
	}
}

func TestLoadSaveRejectsUnknownContainer(t *testing.T) {
	unknown := make([]byte, 0x100)
	copy(unknown, []byte("NOPE"))

	cases := map[string]string{
		"unknown magic":        writeFixture(t, "unknown.sl2", unknown, pcFixtureSize),
		"shorter than a magic": writeFixture(t, "tiny.sl2", []byte{0xCB, 0x01}, 2),
		"empty file":           writeFixture(t, "empty.sl2", nil, 0),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New().LoadSave(path, ""); err == nil {
				t.Fatal("LoadSave accepted an unsupported container")
			}
		})
	}
}

func TestLoadSaveRejectsTruncatedContainer(t *testing.T) {
	cases := []struct {
		name   string
		header []byte
		size   int64
	}{
		{"PC missing one byte of UserData10", pcHeader(), pcFixtureSize - 1},
		{"PC header only", pcHeader(), pcHeaderSize},
		{"PC shorter than its header", pcHeader(), pcHeaderSize - 1},
		{"PS4 missing one byte of UserData10", ps4Header(), ps4FixtureSize - 1},
		{"PS4 header only", ps4Header(), ps4HeaderSize},
		{"PS4 shorter than its header", ps4Header(), ps4HeaderSize - 1},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeFixture(t, "truncated.sl2", testCase.header, testCase.size)
			if _, err := New().LoadSave(path, ""); err == nil {
				t.Fatal("LoadSave accepted a truncated container")
			}
		})
	}
}

func TestLoadSaveRejectsInconsistentPCEntryCount(t *testing.T) {
	header := pcHeader()
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount-1)
	path := writeFixture(t, "pc.sl2", header, pcFixtureSize)

	if _, err := New().LoadSave(path, ""); err == nil {
		t.Fatal("LoadSave accepted a PC container declaring the wrong entry count")
	}
}

func TestLoadSaveRejectsInconsistentPS4Header(t *testing.T) {
	header := ps4Header()
	binary.LittleEndian.PutUint32(header[ps4EntryTableOffset:], 0)
	path := writeFixture(t, "ps4.sl2", header, ps4FixtureSize)

	if _, err := New().LoadSave(path, ""); err == nil {
		t.Fatal("LoadSave accepted a PS4 container with an inconsistent header")
	}
}

func TestLoadSaveRejectsPlatformMismatch(t *testing.T) {
	cases := []struct {
		name     string
		header   []byte
		size     int64
		expected string
	}{
		{"PS4 file expected to be PC", ps4Header(), ps4FixtureSize, string(PlatformPC)},
		{"PC file expected to be PS4", pcHeader(), pcFixtureSize, string(PlatformPS4)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeFixture(t, "save.sl2", testCase.header, testCase.size)
			engine := New()

			info, err := engine.LoadSave(path, testCase.expected)
			if err == nil {
				t.Fatal("LoadSave accepted a save of the wrong platform")
			}
			if info.SaveSessionID != "" {
				t.Errorf("a session was created despite the mismatch: %q", info.SaveSessionID)
			}
			if len(engine.sessions) != 0 {
				t.Errorf("engine holds %d sessions after a rejected load, want 0", len(engine.sessions))
			}
		})
	}
}

func TestLoadSaveAcceptsMatchingExpectedPlatform(t *testing.T) {
	pcPath := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	if _, err := New().LoadSave(pcPath, string(PlatformPC)); err != nil {
		t.Errorf("LoadSave(pc): %v", err)
	}
	ps4Path := writeFixture(t, "ps4.sl2", ps4Header(), ps4FixtureSize)
	if _, err := New().LoadSave(ps4Path, string(PlatformPS4)); err != nil {
		t.Errorf("LoadSave(ps4): %v", err)
	}
}

func TestLoadSaveRejectsUnknownExpectedPlatform(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)

	for _, expected := range []string{"PC", "pc ", " pc", "ps5", "playstation"} {
		t.Run(expected, func(t *testing.T) {
			if _, err := New().LoadSave(path, expected); err == nil {
				t.Fatalf("LoadSave accepted expectedPlatform %q", expected)
			}
		})
	}
}

func TestLoadSaveRejectsDirectorySource(t *testing.T) {
	if _, err := New().LoadSave(t.TempDir(), ""); err == nil {
		t.Fatal("LoadSave accepted a directory as a save source")
	}
}

func TestGetSessionInfoReturnsTheMetadataOfALoadedSession(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info != loaded {
		t.Errorf("GetSessionInfo = %+v, want the metadata LoadSave returned %+v", info, loaded)
	}
}

func TestGetSessionInfoRejectsEmptyIdentifier(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	if _, err := engine.LoadSave(path, ""); err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	info, err := engine.GetSessionInfo("")
	if err == nil {
		t.Fatal("GetSessionInfo accepted an empty saveSessionID")
	}
	if info != (SessionInfo{}) {
		t.Errorf("GetSessionInfo returned %+v for an empty identifier, want the zero value", info)
	}
}

func TestGetSessionInfoRejectsUnknownIdentifier(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Neither a foreign identifier nor a near miss of the real one resolves: the
	// identifier is matched exactly and never trimmed or normalised.
	unknown := []string{
		"00000000000000000000000000000000",
		" " + loaded.SaveSessionID,
		loaded.SaveSessionID + " ",
		strings.ToUpper(loaded.SaveSessionID),
	}
	for _, saveSessionID := range unknown {
		t.Run(saveSessionID, func(t *testing.T) {
			if _, err := engine.GetSessionInfo(saveSessionID); err == nil {
				t.Fatalf("GetSessionInfo accepted the unknown identifier %q", saveSessionID)
			}
		})
	}
	if len(engine.sessions) != 1 {
		t.Errorf("engine holds %d sessions after rejected lookups, want 1", len(engine.sessions))
	}
}

func TestGetSessionInfoReturnsAnIndependentValue(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	mutated, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	mutated.SaveSessionID = "tampered"
	mutated.Platform = "tampered"
	mutated.Format = "tampered"
	mutated.UnsavedChanges = true

	again, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("second GetSessionInfo: %v", err)
	}
	if again != loaded {
		t.Errorf("changing the returned value changed the stored metadata: %+v, want %+v", again, loaded)
	}
}

func TestGetSessionInfoNeedsNoSourceFileAndExposesNoSnapshot(t *testing.T) {
	path := writeFixture(t, "pc.sl2", pcHeader(), pcFixtureSize)
	engine := New()

	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	// The session exists, so the source is no longer needed: the metadata comes
	// from the session, not from the file.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove source: %v", err)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info != loaded {
		t.Errorf("GetSessionInfo = %+v, want %+v", info, loaded)
	}
	if fields := reflect.TypeOf(info).NumField(); fields != 4 {
		t.Errorf("SessionInfo has %d fields, want the 4 metadata fields", fields)
	}
	for name, value := range map[string]string{
		"saveSessionID": info.SaveSessionID,
		"platform":      info.Platform,
		"format":        info.Format,
	} {
		if strings.Contains(value, path) || strings.Contains(value, string(pcMagic)) {
			t.Errorf("%s = %q exposes the source path or the snapshot content", name, value)
		}
	}
	// The snapshot stays private and untouched behind the session.
	if stored := engine.sessions[loaded.SaveSessionID]; stored == nil || stored.snapshot == nil {
		t.Fatal("GetSessionInfo changed the session or its snapshot")
	}
}

func TestCloseSessionRemovesTheSessionFromTheEngine(t *testing.T) {
	path := writeFixture(t, "close.sl2", pcHeader(), pcFixtureSize)
	engine := New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if err := engine.CloseSession(loaded.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	// The map entry is gone, so the engine holds no reference to the session or
	// to its private snapshot any more.
	if _, exists := engine.sessions[loaded.SaveSessionID]; exists {
		t.Fatal("CloseSession left the session in the engine")
	}
	if _, err := engine.GetSessionInfo(loaded.SaveSessionID); err == nil {
		t.Fatal("GetSessionInfo resolved a closed session")
	}
}

func TestCloseSessionKeepsOtherSessions(t *testing.T) {
	engine := New()
	first, err := engine.LoadSave(writeFixture(t, "first.sl2", pcHeader(), pcFixtureSize), "")
	if err != nil {
		t.Fatalf("LoadSave first: %v", err)
	}
	second, err := engine.LoadSave(writeFixture(t, "second.sl2", pcHeader(), pcFixtureSize), "")
	if err != nil {
		t.Fatalf("LoadSave second: %v", err)
	}

	if err := engine.CloseSession(first.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	info, err := engine.GetSessionInfo(second.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo of the untouched session: %v", err)
	}
	if info != second {
		t.Errorf("GetSessionInfo = %+v, want %+v", info, second)
	}
}

func TestCloseSessionRejectsEmptyIdentifier(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeFixture(t, "empty-id.sl2", pcHeader(), pcFixtureSize), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if err := engine.CloseSession(""); err == nil {
		t.Fatal("CloseSession accepted an empty identifier")
	}
	if _, exists := engine.sessions[loaded.SaveSessionID]; !exists {
		t.Fatal("a rejected CloseSession removed a session")
	}
}

func TestCloseSessionRejectsUnknownIdentifier(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeFixture(t, "unknown-id.sl2", pcHeader(), pcFixtureSize), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// A foreign identifier and a near miss of the real one are both unknown: the
	// identifier is matched exactly, without trimming, normalising or guessing.
	for _, saveSessionID := range []string{
		"00000000000000000000000000000000",
		loaded.SaveSessionID + " ",
		" " + loaded.SaveSessionID,
	} {
		t.Run(saveSessionID, func(t *testing.T) {
			if err := engine.CloseSession(saveSessionID); err == nil {
				t.Fatalf("CloseSession accepted the unknown identifier %q", saveSessionID)
			}
			if _, exists := engine.sessions[loaded.SaveSessionID]; !exists {
				t.Fatal("a rejected CloseSession removed the existing session")
			}
		})
	}

	// Closing twice is rejected the second time: a closed session is never
	// resolved again.
	if err := engine.CloseSession(loaded.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if err := engine.CloseSession(loaded.SaveSessionID); err == nil {
		t.Fatal("CloseSession accepted an already closed session")
	}
}
