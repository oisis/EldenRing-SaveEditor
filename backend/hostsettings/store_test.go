package hostsettings

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStorePersistsTheCompleteSettingsValue covers the round trip a host makes:
// defaults before anything is stored, the stored value after a write, and the
// same value again from a second store reading the same directory.
func TestStorePersistsTheCompleteSettingsValue(t *testing.T) {
	directory := t.TempDir()
	store := NewStore(directory)

	settings, err := store.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if settings != Defaults() {
		t.Fatalf("a host that stored nothing reports %+v, want %+v", settings, Defaults())
	}

	stored, err := store.Set(true, string(RemoteBackupAlways))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := Settings{
		SkipReviewForNormalRisk: true,
		RemoteBackupPolicy:      RemoteBackupAlways,
	}
	if stored != want {
		t.Fatalf("Set reported %+v, want %+v", stored, want)
	}

	reopened, err := NewStore(directory).Get()
	if err != nil {
		t.Fatalf("Get from a second store: %v", err)
	}
	if reopened != want {
		t.Fatalf("a second store reports %+v, want %+v", reopened, want)
	}
}

// TestStoreRefusesAnUnknownPolicyAndInvalidState states the fail-closed half:
// an unknown policy is rejected rather than mapped onto the default, and a
// settings file that cannot be understood is an error rather than a silent
// reset to defaults.
func TestStoreRefusesAnUnknownPolicyAndInvalidState(t *testing.T) {
	directory := t.TempDir()
	store := NewStore(directory)

	if _, err := store.Set(false, "never"); err == nil {
		t.Fatal("Set accepted a policy that would disable the mandatory backup")
	}
	if _, err := store.Set(false, ""); err == nil {
		t.Fatal("Set accepted an empty policy")
	}
	// The refused writes stored nothing at all.
	if _, err := os.Stat(filepath.Join(directory, "host-settings.json")); err == nil {
		t.Fatal("a refused Set still wrote the settings file")
	}

	if err := os.WriteFile(
		filepath.Join(directory, "host-settings.json"),
		[]byte(`{"schemaVersion":99,"debugMode":true}`), 0o600); err != nil {
		t.Fatalf("write the corrupt settings file: %v", err)
	}
	if _, err := NewStore(directory).Get(); err == nil {
		t.Fatal("Get silently accepted a settings file of an unsupported version")
	}
}

// TestInMemoryStoreReportsDefaultsWithoutWriting covers the mode the bridge is
// exercised in: no state directory, no file, and a truthful answer.
func TestInMemoryStoreReportsDefaultsWithoutWriting(t *testing.T) {
	store := NewStore("")
	if store.Directory() != "" {
		t.Fatalf("a store with no directory reports %q", store.Directory())
	}
	stored, err := store.Set(false, string(RemoteBackupAsk))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if stored.SkipReviewForNormalRisk || stored.RemoteBackupPolicy != RemoteBackupAsk {
		t.Fatalf("Set reported %+v", stored)
	}
}
