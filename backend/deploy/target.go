package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	appDirName    = "EldenRing-SaveEditor"
	targetsFile   = "targets.json"
	DefaultPort   = 22
	DefaultSavePath = "/home/deck/.local/share/Steam/steamapps/compatdata/1245620/pfx/drive_c/users/steamuser/AppData/Roaming/EldenRing/{STEAM_ID}/ER0000.sl2"
	DefaultStartCmd = "steam steam://rungameid/1245620"
	DefaultStopCmd  = "pkill -TERM -f eldenring.exe"
)

// Target type constants.
const (
	TargetTypeSSH   = "ssh"
	TargetTypeLocal = "local"
)

// Target represents a deploy destination — either a remote SSH host or a local directory.
type Target struct {
	Type         string `json:"type"` // "ssh" or "local"
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	KeyPath      string `json:"keyPath"`
	SavePath     string `json:"savePath"`
	GameStartCmd string `json:"gameStartCmd"`
	GameStopCmd  string `json:"gameStopCmd"`
}

// IsLocal returns true if this is a local (non-SSH) target.
func (t Target) IsLocal() bool {
	return t.Type == TargetTypeLocal
}

// TargetStore manages persistent storage of deploy targets.
type TargetStore struct {
	mu      sync.Mutex
	targets []Target
	path    string
}

// NewTargetStore creates a store that reads/writes targets.json in the app config directory.
func NewTargetStore() (*TargetStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine config dir: %w", err)
	}
	dir := filepath.Join(configDir, appDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create config dir: %w", err)
	}
	s := &TargetStore{path: filepath.Join(dir, targetsFile)}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

// List returns all configured targets.
func (s *TargetStore) List() []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Target, len(s.targets))
	copy(out, s.targets)
	return out
}

// Get returns a target by name.
func (s *TargetStore) Get(name string) (Target, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.targets {
		if t.Name == name {
			return t, true
		}
	}
	return Target{}, false
}

// Save adds or updates a target and persists to disk.
func (s *TargetStore) Save(t Target) error {
	if t.Name == "" {
		return fmt.Errorf("target name is required")
	}
	if t.Type == "" {
		t.Type = TargetTypeSSH
	}
	if t.Type == TargetTypeSSH && t.Host == "" {
		return fmt.Errorf("host is required for SSH targets")
	}
	if t.SavePath == "" {
		return fmt.Errorf("save path is required")
	}
	if t.Port == 0 {
		t.Port = DefaultPort
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i, existing := range s.targets {
		if existing.Name == t.Name {
			s.targets[i] = t
			found = true
			break
		}
	}
	if !found {
		s.targets = append(s.targets, t)
	}
	return s.persist()
}

// Delete removes a target by name and persists to disk.
func (s *TargetStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.targets {
		if t.Name == name {
			s.targets = append(s.targets[:i], s.targets[i+1:]...)
			return s.persist()
		}
	}
	return fmt.Errorf("target %q not found", name)
}

func (s *TargetStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.targets)
}

func (s *TargetStore) persist() error {
	data, err := json.MarshalIndent(s.targets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}
