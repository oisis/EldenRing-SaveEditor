package savesession

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	closeSaveHeaderSize       = 0x300
	closeSaveEntryCountOffset = 0x0C
	closeSaveEntryCount       = 12
	closeSaveFixtureSize      = int64(closeSaveHeaderSize) + 10*0x280010 + 0x60010
)

// writeCloseSaveFixture writes a minimal synthetic PC save into t.TempDir() and
// returns its path.
func writeCloseSaveFixture(t *testing.T) string {
	t.Helper()

	header := make([]byte, closeSaveHeaderSize)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[closeSaveEntryCountOffset:], closeSaveEntryCount)

	path := filepath.Join(t.TempDir(), "close-save.sl2")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(header); err != nil {
		t.Fatalf("write fixture header: %v", err)
	}
	if err := file.Truncate(closeSaveFixtureSize); err != nil {
		t.Fatalf("size fixture: %v", err)
	}
	return path
}

// loadCloseSaveSession creates the session this endpoint closes.
func loadCloseSaveSession(t *testing.T) (*saveengine.Engine, LoadSaveResult, string) {
	t.Helper()

	path := writeCloseSaveFixture(t)
	engine := saveengine.New()
	loaded, err := LoadSave(engine, path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded, path
}

func TestCloseSaveClosesALoadedSession(t *testing.T) {
	engine, loaded, path := loadCloseSaveSession(t)

	if err := CloseSave(engine, loaded.SaveSessionID); err != nil {
		t.Fatalf("CloseSave: %v", err)
	}
	// The session is gone, so SaveEngine no longer resolves it. The check reads
	// SaveEngine directly instead of going through another endpoint.
	if _, err := engine.GetSessionInfo(loaded.SaveSessionID); err == nil {
		t.Fatal("SaveEngine resolved a closed session")
	}
	// The source file is untouched: closing releases memory, not the save.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat source after CloseSave: %v", err)
	}
}

func TestCloseSaveRejectsMissingEngine(t *testing.T) {
	if err := CloseSave(nil, "any-save-session"); err == nil {
		t.Fatal("CloseSave accepted a nil engine")
	}
}
