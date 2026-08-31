package savesession

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic container layouts. The endpoint owns none of these values; they are
// duplicated here only to build fixtures the SaveEngine accepts.
const (
	pcHeaderSize       = 0x300
	pcEntryCountOffset = 0x0C
	pcEntryCount       = 12
	pcFixtureSize      = int64(pcHeaderSize) + 10*0x280010 + 0x60010

	ps4HeaderSize       = 0x70
	ps4EntryTableOffset = 0x10
	ps4EntryCount       = 12
	ps4EntryStride      = 8
	ps4FirstEntryIndex  = 7
	ps4EntryMarker      = 0x7F7F7F7F
	ps4FixtureSize      = int64(ps4HeaderSize) + 10*0x280000 + 0x60000
)

func writeFixture(t *testing.T, header []byte, size int64) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "save.sl2")
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

func writePCFixture(t *testing.T) string {
	t.Helper()

	header := make([]byte, pcHeaderSize)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount)
	return writeFixture(t, header, pcFixtureSize)
}

func writePS4Fixture(t *testing.T) string {
	t.Helper()

	header := make([]byte, ps4HeaderSize)
	copy(header, []byte{0xCB, 0x01, 0x9C, 0x2C})
	for entry := 0; entry < ps4EntryCount; entry++ {
		offset := ps4EntryTableOffset + entry*ps4EntryStride
		binary.LittleEndian.PutUint32(header[offset:], uint32(ps4FirstEntryIndex+entry))
		binary.LittleEndian.PutUint32(header[offset+4:], ps4EntryMarker)
	}
	return writeFixture(t, header, ps4FixtureSize)
}

func fingerprint(t *testing.T, path string) ([32]byte, os.FileInfo) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}
	return sha256.Sum256(content), info
}

func TestLoadSaveCreatesSessionForPCSave(t *testing.T) {
	result, err := LoadSave(saveengine.New(), writePCFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if result.SaveSessionID == "" {
		t.Error("saveSessionID is empty")
	}
	if result.Platform != "pc" {
		t.Errorf("platform = %q, want %q", result.Platform, "pc")
	}
	if result.Format == "" {
		t.Error("format is empty")
	}
	if result.UnsavedChanges {
		t.Error("a freshly loaded session reports unsaved changes")
	}
}

func TestLoadSaveCreatesSessionForPS4Save(t *testing.T) {
	result, err := LoadSave(saveengine.New(), writePS4Fixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if result.SaveSessionID == "" {
		t.Error("saveSessionID is empty")
	}
	if result.Platform != "ps4" {
		t.Errorf("platform = %q, want %q", result.Platform, "ps4")
	}
	if result.UnsavedChanges {
		t.Error("a freshly loaded session reports unsaved changes")
	}
}

func TestLoadSaveRejectsUnknownContainer(t *testing.T) {
	unknown := make([]byte, 0x100)
	copy(unknown, []byte("NOPE"))

	if _, err := LoadSave(saveengine.New(), writeFixture(t, unknown, pcFixtureSize), "", "local"); err == nil {
		t.Fatal("LoadSave accepted an unsupported container")
	}
}

func TestLoadSaveRejectsTruncatedContainer(t *testing.T) {
	header := make([]byte, pcHeaderSize)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount)

	if _, err := LoadSave(saveengine.New(), writeFixture(t, header, pcFixtureSize-1), "", "local"); err == nil {
		t.Fatal("LoadSave accepted a truncated PC container")
	}
}

func TestLoadSaveRejectsExpectedPlatformMismatch(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		expected string
	}{
		{"PS4 save expected to be PC", writePS4Fixture(t), "pc"},
		{"PC save expected to be PS4", writePCFixture(t), "ps4"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := LoadSave(saveengine.New(), testCase.path, testCase.expected, "local")
			if err == nil {
				t.Fatal("LoadSave accepted a save of the wrong platform")
			}
			if result.SaveSessionID != "" {
				t.Errorf("a session was created despite the mismatch: %q", result.SaveSessionID)
			}
		})
	}
}

func TestLoadSaveAcceptsMatchingExpectedPlatform(t *testing.T) {
	if _, err := LoadSave(saveengine.New(), writePCFixture(t), "pc", "local"); err != nil {
		t.Errorf("LoadSave(pc): %v", err)
	}
	if _, err := LoadSave(saveengine.New(), writePS4Fixture(t), "ps4", "local"); err != nil {
		t.Errorf("LoadSave(ps4): %v", err)
	}
}

func TestLoadSaveRejectsUnknownExpectedPlatform(t *testing.T) {
	path := writePCFixture(t)

	for _, expected := range []string{"PC", "pc ", " pc", "ps5", "unknown"} {
		t.Run(expected, func(t *testing.T) {
			result, err := LoadSave(saveengine.New(), path, expected, "local")
			if err == nil {
				t.Fatalf("LoadSave accepted expectedPlatform %q", expected)
			}
			if result.SaveSessionID != "" {
				t.Errorf("a session was created for expectedPlatform %q", expected)
			}
		})
	}
}

// TestLoadSaveReportsTheExactSourceForBothPlatforms covers the source metadata
// and the initial revision the endpoint now carries, on PC and on PS4 alike and
// for both accepted source kinds. The path is checked for exact equality, so a
// trimming, recasing or resolving implementation cannot pass.
func TestLoadSaveReportsTheExactSourceForBothPlatforms(t *testing.T) {
	for platform, writeFixture := range map[string]func(*testing.T) string{
		"pc":  writePCFixture,
		"ps4": writePS4Fixture,
	} {
		for _, sourceKind := range []string{"local", "temporary"} {
			t.Run(platform+"/"+sourceKind, func(t *testing.T) {
				path := writeFixture(t)

				result, err := LoadSave(saveengine.New(), path, "", sourceKind)
				if err != nil {
					t.Fatalf("LoadSave: %v", err)
				}
				if result.Platform != platform {
					t.Errorf("platform = %q, want %q", result.Platform, platform)
				}
				if result.SourcePath != path {
					t.Errorf("sourcePath = %q, want the exact source %q", result.SourcePath, path)
				}
				if result.SourceKind != sourceKind {
					t.Errorf("sourceKind = %q, want %q", result.SourceKind, sourceKind)
				}
				if result.SaveRevision != "0" {
					t.Errorf("saveRevision = %q, want %q", result.SaveRevision, "0")
				}
				if result.UnsavedChanges {
					t.Error("a freshly loaded session reports unsaved changes")
				}
			})
		}
	}
}

// TestLoadSaveRejectsUnknownSourceKind proves the endpoint adds no default,
// alias or normalisation of its own, and that a rejected value leaves no
// session behind for the engine it was given.
func TestLoadSaveRejectsUnknownSourceKind(t *testing.T) {
	for _, sourceKind := range []string{"", " ", "Local", "local ", "temp", "remote"} {
		t.Run(sourceKind, func(t *testing.T) {
			engine := saveengine.New()

			result, err := LoadSave(engine, writePCFixture(t), "", sourceKind)
			if err == nil {
				t.Fatalf("LoadSave accepted the unknown source kind %q", sourceKind)
			}
			if result != (LoadSaveResult{}) {
				t.Errorf("LoadSave returned %+v for a rejected source kind, want the zero value", result)
			}
			// No session exists, so nothing can be read back under any identifier
			// the rejected call might have created.
			if _, err := GetLoadedSave(engine, result.SaveSessionID); err == nil {
				t.Error("a rejected source kind left a readable session behind")
			}
		})
	}
}

func TestLoadSaveRejectsMissingEngine(t *testing.T) {
	if _, err := LoadSave(nil, writePCFixture(t), "", "local"); err == nil {
		t.Fatal("LoadSave accepted a missing engine")
	}
}

func TestLoadSaveDoesNotModifySource(t *testing.T) {
	path := writePCFixture(t)
	digestBefore, infoBefore := fingerprint(t, path)

	if _, err := LoadSave(saveengine.New(), path, "pc", "local"); err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	digestAfter, infoAfter := fingerprint(t, path)
	if digestAfter != digestBefore {
		t.Error("LoadSave changed the content of the source file")
	}
	if infoAfter.Size() != infoBefore.Size() {
		t.Errorf("source size changed: %d -> %d", infoBefore.Size(), infoAfter.Size())
	}
	if !infoAfter.ModTime().Equal(infoBefore.ModTime()) {
		t.Errorf("source modification time changed: %v -> %v", infoBefore.ModTime(), infoAfter.ModTime())
	}
}
