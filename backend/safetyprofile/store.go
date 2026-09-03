package safetyprofile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store is the host-local persistence of the global safety profile. It is an
// application setting, so it lives beside the other host settings of the
// application data directory and never inside a save session, a save snapshot
// or SaveEngine.
//
// An empty state directory selects an in-memory store. That keeps package users
// and tests from writing host state unless the composition root explicitly
// enables persistence, exactly like SaveEngine's own state directory.
type Store struct {
	mutex     sync.Mutex
	directory string
	profile   Profile
	loaded    bool
}

const settingsSchemaVersion = 1

type settingsFile struct {
	SchemaVersion int     `json:"schemaVersion"`
	SafetyProfile Profile `json:"safetyProfile"`
}

// NewStore returns a store backed by directory. Constructing it is side-effect
// free: the file is read on the first Get and written only by Set.
func NewStore(directory string) *Store {
	return &Store{directory: directory, profile: Default}
}

func (store *Store) path() string {
	if store.directory == "" {
		return ""
	}
	return filepath.Join(store.directory, "application-settings.json")
}

// Get returns the active profile, reading the stored one on first use. A host
// that never stored a profile runs under Default; a stored file that cannot be
// read or does not carry exactly one of the three known values is an error and
// never silently replaced by the default.
func (store *Store) Get() (Profile, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return "", err
	}
	return store.profile, nil
}

// Set validates value, stores it and returns the profile now in effect. A
// failed write leaves the previous profile in memory as well as on disk.
func (store *Store) Set(value string) (Profile, error) {
	profile, err := Parse(value)
	if err != nil {
		return "", err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return "", err
	}
	previous := store.profile
	store.profile = profile
	if err := store.persistLocked(); err != nil {
		store.profile = previous
		return "", err
	}
	return store.profile, nil
}

func (store *Store) loadLocked() error {
	if store.loaded {
		return nil
	}
	store.profile = Default
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
		return fmt.Errorf("cannot read application settings: %w", err)
	}
	var stored settingsFile
	if err := json.Unmarshal(data, &stored); err != nil ||
		stored.SchemaVersion != settingsSchemaVersion {
		return errors.New("application settings are invalid")
	}
	profile, err := Parse(string(stored.SafetyProfile))
	if err != nil {
		return fmt.Errorf("application settings are invalid: %w", err)
	}
	store.profile = profile
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
		SafetyProfile: store.profile,
	})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create the application settings directory: %w", err)
	}
	// The replacement is atomic: a crash leaves either the previous complete
	// document or the new complete one, never a half-written setting.
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("cannot write application settings: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("cannot store application settings: %w", err)
	}
	return nil
}
