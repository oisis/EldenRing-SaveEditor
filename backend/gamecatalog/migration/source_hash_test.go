package migration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashRelativeFilesChangesWithSourceContent(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "data.go")
	if err := os.WriteFile(filename, []byte("package data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := hashRelativeFiles(root, []string{"data.go"})
	if err != nil {
		t.Fatalf("first hashRelativeFiles: %v", err)
	}
	if err := os.WriteFile(filename, []byte("package data\nvar Version = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := hashRelativeFiles(root, []string{"data.go"})
	if err != nil {
		t.Fatalf("second hashRelativeFiles: %v", err)
	}
	if first == second {
		t.Fatalf("source hash did not change after file-content change: %s", first)
	}
}

func TestLegacySourceVersionHashesExactSnapshot(t *testing.T) {
	snapshot := collectLegacySnapshot()
	first, err := hashLegacySnapshot(snapshot)
	if err != nil {
		t.Fatalf("first hashLegacySnapshot: %v", err)
	}
	snapshot.Items[0].Name += " changed"
	second, err := hashLegacySnapshot(snapshot)
	if err != nil {
		t.Fatalf("second hashLegacySnapshot: %v", err)
	}
	if first == second {
		t.Fatalf("legacy snapshot hash did not change after source-data change: %s", first)
	}
}
