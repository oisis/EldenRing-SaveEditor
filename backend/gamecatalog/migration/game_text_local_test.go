package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func readLocalGameTextFixture(t *testing.T) *GameTextData {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	gameTextDirectory := filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "..",
		"tmp", "regulation-bin-dump", "msg", "fmg-extracted",
	)
	if _, err := os.Stat(gameTextDirectory); err != nil {
		if os.IsNotExist(err) {
			t.Skip("local proprietary game-text fixture is not available")
		}
		t.Fatalf("stat game-text fixture: %v", err)
	}
	gameText, err := ReadGameTextDirectory(gameTextDirectory)
	if err != nil {
		t.Fatalf("ReadGameTextDirectory: %v", err)
	}
	return gameText
}
