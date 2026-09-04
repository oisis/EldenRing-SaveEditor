// Package hostsettings owns the persistent host application settings that are
// not save state: the two Save behavior preferences of Tools -> Settings.
//
// It is the single backend source of truth for those values. React caches them
// through React Query and never keeps a second authoritative copy, and no save
// session, snapshot or recovery journal ever carries them.
package hostsettings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RemoteBackupPolicy states how a deployment that would replace an existing
// target save asks about its mandatory backup. Neither value can disable the
// backup itself: deployment.md section 5 states that a "never backup" mode does
// not exist, so the vocabulary deliberately has no third member.
type RemoteBackupPolicy string

const (
	// RemoteBackupAsk confirms the mandatory backup before every replacement.
	RemoteBackupAsk RemoteBackupPolicy = "ask"
	// RemoteBackupAlways creates the mandatory backup without asking again.
	RemoteBackupAlways RemoteBackupPolicy = "always"
)

// DefaultRemoteBackupPolicy is what a host that never stored a preference runs
// under: deployment.md states the application asks by default.
const DefaultRemoteBackupPolicy = RemoteBackupAsk

// RemoteBackupPolicies is the closed vocabulary the store accepts, in the order
// the interface presents it.
func RemoteBackupPolicies() []RemoteBackupPolicy {
	return []RemoteBackupPolicy{RemoteBackupAsk, RemoteBackupAlways}
}

// ParseRemoteBackupPolicy validates one stated policy. An unknown value is
// rejected rather than silently mapped onto the default: a misspelled policy
// must never quietly change how a deployment backs up an existing target.
func ParseRemoteBackupPolicy(value string) (RemoteBackupPolicy, error) {
	switch RemoteBackupPolicy(value) {
	case RemoteBackupAsk:
		return RemoteBackupAsk, nil
	case RemoteBackupAlways:
		return RemoteBackupAlways, nil
	}
	return "", fmt.Errorf(
		"unknown remote backup policy %q; expected %q or %q",
		value, RemoteBackupAsk, RemoteBackupAlways)
}

// Settings is the complete set of host settings this package owns.
type Settings struct {
	// SkipReviewForNormalRisk allows Save and deployment to go straight to the
	// operation when the completed validation reports no warning, no ban risk
	// and no critical finding. It never skips the validation itself.
	SkipReviewForNormalRisk bool `json:"skipReviewForNormalRisk"`
	// RemoteBackupPolicy selects Ask or Always for the mandatory backup of an
	// existing deployment target.
	RemoteBackupPolicy RemoteBackupPolicy `json:"remoteBackupPolicy"`
}

// Defaults is the state of a host that stored nothing yet.
func Defaults() Settings {
	return Settings{RemoteBackupPolicy: DefaultRemoteBackupPolicy}
}

const settingsSchemaVersion = 1

type settingsFile struct {
	SchemaVersion int `json:"schemaVersion"`
	Settings
}

// Store persists Settings in the host state directory supplied by the
// composition root. An empty directory selects the package's in-memory mode:
// nothing is written and the defaults are reported truthfully, which is what
// keeps the bridge exercisable without a host.
type Store struct {
	mutex     sync.Mutex
	directory string
	settings  Settings
	loaded    bool
}

// NewStore returns a store backed by directory.
func NewStore(directory string) *Store {
	return &Store{directory: directory, settings: Defaults()}
}

// Directory reports the host state directory the store was built with. It is
// the configuration directory the Settings screen offers to open, and it comes
// from the composition root rather than from any frontend argument.
func (store *Store) Directory() string {
	if store == nil {
		return ""
	}
	return store.directory
}

func (store *Store) path() string {
	if store.directory == "" {
		return ""
	}
	return filepath.Join(store.directory, "host-settings.json")
}

// Get reports the stored settings.
func (store *Store) Get() (Settings, error) {
	if store == nil {
		return Defaults(), nil
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return Settings{}, err
	}
	return store.settings, nil
}

// Set stores a complete new settings value and returns what is now in effect.
// The policy is validated before anything is written, and a failed write
// restores the previous in-memory value so the store never reports a setting
// the host did not actually keep.
func (store *Store) Set(
	skipReviewForNormalRisk bool,
	remoteBackupPolicy string,
) (Settings, error) {
	if store == nil {
		return Settings{}, errors.New("host settings store is not available")
	}
	policy, err := ParseRemoteBackupPolicy(remoteBackupPolicy)
	if err != nil {
		return Settings{}, err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return Settings{}, err
	}
	previous := store.settings
	store.settings = Settings{
		SkipReviewForNormalRisk: skipReviewForNormalRisk,
		RemoteBackupPolicy:      policy,
	}
	if err := store.persistLocked(); err != nil {
		store.settings = previous
		return Settings{}, err
	}
	return store.settings, nil
}

func (store *Store) loadLocked() error {
	if store.loaded {
		return nil
	}
	store.settings = Defaults()
	path := store.path()
	if path == "" {
		store.loaded = true
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		store.loaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read host settings: %w", err)
	}
	var stored settingsFile
	if err := json.Unmarshal(data, &stored); err != nil ||
		stored.SchemaVersion != settingsSchemaVersion {
		return errors.New("host settings are invalid")
	}
	policy, err := ParseRemoteBackupPolicy(string(stored.RemoteBackupPolicy))
	if err != nil {
		return fmt.Errorf("host settings are invalid: %w", err)
	}
	store.settings = stored.Settings
	store.settings.RemoteBackupPolicy = policy
	store.loaded = true
	return nil
}

func (store *Store) persistLocked() error {
	path := store.path()
	if path == "" {
		return nil
	}
	data, err := json.Marshal(settingsFile{
		SchemaVersion: settingsSchemaVersion,
		Settings:      store.settings,
	})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create the host settings directory: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("cannot write host settings: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("cannot store host settings: %w", err)
	}
	return nil
}
