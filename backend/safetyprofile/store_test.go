package safetyprofile

import (
	"os"
	"path/filepath"
	"testing"
)

// The store is the one place the global profile is persisted. A host that never
// stored one runs under the product default, a stored value survives a restart,
// and an unusable stored document is an error rather than a silent default.

func TestStorePersistsTheProfileAcrossRestarts(t *testing.T) {
	directory := t.TempDir()

	store := NewStore(directory)
	profile, err := store.Get()
	if err != nil {
		t.Fatalf("Get on a fresh host: %v", err)
	}
	if profile != Default {
		t.Errorf("a host that never stored a profile runs under %q, want %q", profile, Default)
	}

	stored, err := store.Set(string(Chaos))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if stored != Chaos {
		t.Errorf("Set returned %q, want %q", stored, Chaos)
	}

	// A second store over the same directory is what a restarted host builds.
	restarted, err := NewStore(directory).Get()
	if err != nil {
		t.Fatalf("Get after a restart: %v", err)
	}
	if restarted != Chaos {
		t.Errorf("the restarted host runs under %q, want %q", restarted, Chaos)
	}
}

// An unknown value is refused before anything is written, so the profile in
// effect is unchanged in memory and on disk.
func TestStoreRejectsAnUnknownProfileAndKeepsThePreviousOne(t *testing.T) {
	directory := t.TempDir()
	store := NewStore(directory)
	if _, err := store.Set(string(ExpandedLimits)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	for _, value := range []string{"", "Safe", "safe_mode", "chaos "} {
		if _, err := store.Set(value); err == nil {
			t.Errorf("Set(%q) was accepted, want a rejection", value)
		}
	}

	profile, err := store.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if profile != ExpandedLimits {
		t.Errorf("the refused writes changed the profile to %q, want %q", profile, ExpandedLimits)
	}
	if restarted, err := NewStore(directory).Get(); err != nil || restarted != ExpandedLimits {
		t.Errorf("the stored document reads back as %q (%v), want %q",
			restarted, err, ExpandedLimits)
	}
}

// A stored document that cannot be understood is reported instead of being
// replaced by the default: a host must never run under a profile the user did
// not choose because a file became unreadable.
func TestStoreReportsAnUnusableStoredDocument(t *testing.T) {
	for name, content := range map[string]string{
		"not json":          "{",
		"unknown schema":    `{"schemaVersion":99,"safetyProfile":"safe"}`,
		"unknown profile":   `{"schemaVersion":1,"safetyProfile":"reckless"}`,
		"no profile at all": `{"schemaVersion":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "application-settings.json")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			if profile, err := NewStore(directory).Get(); err == nil {
				t.Errorf("the unusable document was accepted as %q", profile)
			}
		})
	}
}

// An empty state directory selects the in-memory store, so a package user that
// never opted into persistence writes no host state.
func TestStoreWithoutADirectoryWritesNothing(t *testing.T) {
	store := NewStore("")
	if _, err := store.Set(string(Chaos)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	profile, err := store.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if profile != Chaos {
		t.Errorf("the in-memory store lost the profile: %q, want %q", profile, Chaos)
	}
}
