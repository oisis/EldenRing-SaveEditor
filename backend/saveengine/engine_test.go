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
