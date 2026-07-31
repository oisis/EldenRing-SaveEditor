package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func readLocalRegulationParameterFixture(
	t *testing.T,
) *RegulationParameterData {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	directory := filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "..",
		"tmp", "regulation-bin-dump", "params",
	)
	if _, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) {
			t.Skip("local proprietary regulation parameter fixture is unavailable")
		}
		t.Fatalf("stat regulation parameter fixture: %v", err)
	}
	data, err := ReadRegulationParameterDirectory(directory)
	if err != nil {
		t.Fatalf("ReadRegulationParameterDirectory: %v", err)
	}
	return data
}
