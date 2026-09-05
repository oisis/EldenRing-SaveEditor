package saveengine

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRetentionRemovesOnlyBackupsThisApplicationOwns: once the name pattern is
// configurable, a name is no longer proof of ownership. Retention may act on
// what the index records and on the fixed 2.0 grammar, and on nothing else.
func TestRetentionRemovesOnlyBackupsThisApplicationOwns(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "ER0000.sl2")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	owned := map[string]time.Time{}
	write := func(name string) string {
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return path
	}
	// Two backups written under a configured pattern; only the index states that
	// they are ours and when they were taken.
	oldest := write("saveforge-20260905100000-ER0000.sl2_bak")
	owned["saveforge-20260905100000-ER0000.sl2_bak"] = time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	newer := write("saveforge-20260905110000-ER0000.sl2_bak")
	owned["saveforge-20260905110000-ER0000.sl2_bak"] = time.Date(2026, 9, 5, 11, 0, 0, 0, time.UTC)
	// One backup from before the pattern was configurable, recognised by name.
	legacy := write("ER0000.sl2.20260905120000_bak")
	stamp := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(legacy, stamp, stamp); err != nil {
		t.Fatalf("date the legacy backup: %v", err)
	}
	// Two files this application never wrote. They sit in the same directory and
	// one of them even ends in the backup suffix.
	foreign := write("somebody-elses-file_bak")
	unrelated := write("notes.txt")

	removed, reached, err := pruneAutomaticBackups(target, 1, owned)
	if err != nil || !reached {
		t.Fatalf("pruneAutomaticBackups = %v, %v, %v", removed, reached, err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %v, want the two oldest owned backups", removed)
	}
	for _, path := range []string{oldest, newer} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%q survived retention: %v", path, err)
		}
	}
	for _, path := range []string{legacy, foreign, unrelated, target} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%q was removed but is not ours to remove: %v", path, err)
		}
	}
}

// TestBackupIndexSurvivesARestart: the provenance of a custom-named backup is
// the only thing that keeps it visible to retention, so it must be persisted.
func TestBackupIndexSurvivesARestart(t *testing.T) {
	state := t.TempDir()
	target := filepath.Join(t.TempDir(), "ER0000.sl2")
	engine := NewWithOptions(EngineOptions{StateDirectory: state})
	when := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	if err := engine.recordAutomaticBackupLocked(target, "custom_bak", when); err != nil {
		t.Fatalf("recordAutomaticBackupLocked: %v", err)
	}

	restarted := NewWithOptions(EngineOptions{StateDirectory: state})
	owned, err := restarted.ownedAutomaticBackupsLocked(target)
	if err != nil {
		t.Fatalf("ownedAutomaticBackupsLocked: %v", err)
	}
	if recorded, ok := owned["custom_bak"]; !ok || !recorded.Equal(when) {
		t.Fatalf("owned = %v, want the recorded creation time", owned)
	}

	if err := restarted.forgetAutomaticBackupsLocked(target, []string{"custom_bak"}); err != nil {
		t.Fatalf("forgetAutomaticBackupsLocked: %v", err)
	}
	again := NewWithOptions(EngineOptions{StateDirectory: state})
	owned, err = again.ownedAutomaticBackupsLocked(target)
	if err != nil {
		t.Fatalf("ownedAutomaticBackupsLocked: %v", err)
	}
	if len(owned) != 0 {
		t.Fatalf("owned = %v, want the removed backup forgotten", owned)
	}
}
