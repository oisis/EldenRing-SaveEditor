package savesession

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getLoadedSaveHeaderSize       = 0x300
	getLoadedSaveEntryCountOffset = 0x0C
	getLoadedSaveEntryCount       = 12
	getLoadedSaveFixtureSize      = int64(getLoadedSaveHeaderSize) + 10*0x280010 + 0x60010
)

// writeGetLoadedSaveFixture writes a minimal synthetic PC save into t.TempDir()
// and returns its path.
func writeGetLoadedSaveFixture(t *testing.T) string {
	t.Helper()

	header := make([]byte, getLoadedSaveHeaderSize)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[getLoadedSaveEntryCountOffset:], getLoadedSaveEntryCount)

	path := filepath.Join(t.TempDir(), "get-loaded-save.sl2")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(header); err != nil {
		t.Fatalf("write fixture header: %v", err)
	}
	if err := file.Truncate(getLoadedSaveFixtureSize); err != nil {
		t.Fatalf("size fixture: %v", err)
	}
	return path
}

// loadGetLoadedSaveSession creates the session this endpoint reads back.
func loadGetLoadedSaveSession(t *testing.T) (*saveengine.Engine, LoadSaveResult, string) {
	t.Helper()

	path := writeGetLoadedSaveFixture(t)
	engine := saveengine.New()
	loaded, err := LoadSave(engine, path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded, path
}

func TestGetLoadedSaveReturnsTheMetadataOfALoadedSession(t *testing.T) {
	engine, loaded, _ := loadGetLoadedSaveSession(t)

	result, err := GetLoadedSave(engine, loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetLoadedSave: %v", err)
	}
	if result != loaded {
		t.Errorf("GetLoadedSave = %+v, want the session LoadSave created %+v", result, loaded)
	}
}

func TestGetLoadedSaveRejectsMissingEngine(t *testing.T) {
	result, err := GetLoadedSave(nil, "any-save-session")
	if err == nil {
		t.Fatal("GetLoadedSave accepted a nil engine")
	}
	if result != (GetLoadedSaveResult{}) {
		t.Errorf("GetLoadedSave returned %+v for a nil engine, want the zero value", result)
	}
}

func TestGetLoadedSaveRejectsEmptySaveSessionID(t *testing.T) {
	engine, _, _ := loadGetLoadedSaveSession(t)

	result, err := GetLoadedSave(engine, "")
	if err == nil {
		t.Fatal("GetLoadedSave accepted an empty saveSessionID")
	}
	if result != (GetLoadedSaveResult{}) {
		t.Errorf("GetLoadedSave returned %+v for an empty identifier, want the zero value", result)
	}
}

func TestGetLoadedSaveRejectsUnknownSaveSessionID(t *testing.T) {
	engine, loaded, _ := loadGetLoadedSaveSession(t)

	// A foreign identifier and a near miss of the real one are both unknown: the
	// endpoint adds no trimming, normalisation, or guessing of its own.
	for _, saveSessionID := range []string{
		"00000000000000000000000000000000",
		loaded.SaveSessionID + " ",
		" " + loaded.SaveSessionID,
	} {
		t.Run(saveSessionID, func(t *testing.T) {
			result, err := GetLoadedSave(engine, saveSessionID)
			if err == nil {
				t.Fatalf("GetLoadedSave accepted the unknown identifier %q", saveSessionID)
			}
			if result != (GetLoadedSaveResult{}) {
				t.Errorf("GetLoadedSave returned %+v for an unknown identifier, want the zero value", result)
			}
		})
	}
}

func TestGetLoadedSaveResultCarriesOnlySessionMetadata(t *testing.T) {
	engine, loaded, _ := loadGetLoadedSaveSession(t)

	result, err := GetLoadedSave(engine, loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetLoadedSave: %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	want := map[string]any{
		"saveSessionID":  loaded.SaveSessionID,
		"platform":       "pc",
		"format":         "bnd4",
		"unsavedChanges": false,
	}
	if len(fields) != len(want) {
		t.Errorf("result JSON has fields %v, want exactly %v", fields, want)
	}
	for name, expected := range want {
		value, present := fields[name]
		if !present {
			t.Errorf("result JSON is missing %q", name)
			continue
		}
		if value != expected {
			t.Errorf("result JSON %q = %v, want %v", name, value, expected)
		}
	}
}

func TestGetLoadedSaveNeitherLoadsNorOpensTheSaveFile(t *testing.T) {
	engine, loaded, path := loadGetLoadedSaveSession(t)

	// The session already exists, so the endpoint must not need the file: it
	// neither calls LoadSave nor opens anything itself.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove source: %v", err)
	}

	result, err := GetLoadedSave(engine, loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetLoadedSave: %v", err)
	}
	if result != loaded {
		t.Errorf("GetLoadedSave = %+v, want %+v", result, loaded)
	}

	// A second call still resolves the same session, so no new session was
	// created behind the getter.
	again, err := GetLoadedSave(engine, loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("second GetLoadedSave: %v", err)
	}
	if again != loaded {
		t.Errorf("second GetLoadedSave = %+v, want %+v", again, loaded)
	}
}
