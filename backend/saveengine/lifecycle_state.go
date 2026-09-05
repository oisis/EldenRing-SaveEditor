package saveengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
	"github.com/oisis/EldenRing-SaveForge/backend/backupname"
)

const (
	defaultBackupRetention = 10
	maximumRecentFiles     = 10
)

// SaveLifecycleSettings contains the configurable local backup policy: how many
// automatic backups are kept and what they are called.
//
// It is the single owner of the backup name pattern. Deployment target backups
// read the same value through the composition root rather than keeping a second
// copy of it in the host settings.
type SaveLifecycleSettings struct {
	BackupRetention      int  `json:"backupRetention"`
	RetentionNoticeShown bool `json:"retentionNoticeShown"`
	// BackupNamePattern accepts exactly {filename} and {timestamp}, each once,
	// plus safe literal text. A configuration written before this setting existed
	// carries none and keeps the default name.
	BackupNamePattern string `json:"backupNamePattern,omitempty"`
	// BackupNameExample is derived, never configured: it is what the pattern now
	// in effect produces for a sample save, so the Settings screen can show the
	// real name without reimplementing the grammar.
	BackupNameExample string `json:"backupNameExample,omitempty"`
}

type lifecycleSettingsFile struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Settings      SaveLifecycleSettings `json:"settings"`
}

func defaultSaveLifecycleSettings() SaveLifecycleSettings {
	return SaveLifecycleSettings{
		BackupRetention:   defaultBackupRetention,
		BackupNamePattern: backupname.Default,
	}
}

// reported fills in the derived example. The stored value is never trusted for
// it: the example always states what the pattern in effect produces today.
func reported(settings SaveLifecycleSettings) SaveLifecycleSettings {
	settings.BackupNamePattern = backupname.Normalise(settings.BackupNamePattern)
	settings.BackupNameExample = backupname.Example(settings.BackupNamePattern)
	return settings
}

func (engine *Engine) lifecycleSettingsPath() string {
	if engine.stateDirectory == "" {
		return ""
	}
	return filepath.Join(engine.stateDirectory, "save-lifecycle-settings.json")
}

func (engine *Engine) loadLifecycleSettingsLocked() error {
	if engine.lifecycleSettingsLoaded {
		return nil
	}
	engine.lifecycleSettings = defaultSaveLifecycleSettings()
	path := engine.lifecycleSettingsPath()
	if path == "" {
		engine.lifecycleSettingsLoaded = true
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		engine.lifecycleSettingsLoaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read save lifecycle settings: %w", err)
	}
	var stored lifecycleSettingsFile
	if err := json.Unmarshal(data, &stored); err != nil || stored.SchemaVersion != 1 {
		return errors.New("save lifecycle settings are invalid")
	}
	if err := validateBackupRetention(stored.Settings.BackupRetention); err != nil {
		return err
	}
	stored.Settings.BackupNamePattern = backupname.Normalise(stored.Settings.BackupNamePattern)
	if err := backupname.Validate(stored.Settings.BackupNamePattern); err != nil {
		return fmt.Errorf("save lifecycle settings are invalid: %w", err)
	}
	engine.lifecycleSettings = stored.Settings
	engine.lifecycleSettingsLoaded = true
	return nil
}

func validateBackupRetention(value int) error {
	if value < 1 || value > 1000 {
		return fmt.Errorf("backupRetention %d is outside the range 1..1000", value)
	}
	return nil
}

func (engine *Engine) persistLifecycleSettingsLocked() error {
	path := engine.lifecycleSettingsPath()
	if path == "" {
		return nil
	}
	data, err := json.Marshal(lifecycleSettingsFile{
		SchemaVersion: 1,
		Settings:      engine.lifecycleSettings,
	})
	if err != nil {
		return err
	}
	return writePrivateFileAtomically(path, data)
}

func (engine *Engine) GetSaveLifecycleSettings() (SaveLifecycleSettings, error) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadLifecycleSettingsLocked(); err != nil {
		return SaveLifecycleSettings{}, err
	}
	return reported(engine.lifecycleSettings), nil
}

// SetSaveLifecycleSettings stores the complete local backup policy.
//
// The backend is the source of the validation rules: an empty pattern means the
// default, and anything the grammar refuses is rejected here rather than
// sanitised into something that would name a file somewhere else. Changing the
// pattern renames nothing: existing backups keep their names and stay usable.
func (engine *Engine) SetSaveLifecycleSettings(
	backupRetention int,
	backupNamePattern string,
) (SaveLifecycleSettings, error) {
	if err := validateBackupRetention(backupRetention); err != nil {
		return SaveLifecycleSettings{}, err
	}
	pattern := backupname.Normalise(backupNamePattern)
	if err := backupname.Validate(pattern); err != nil {
		return SaveLifecycleSettings{}, err
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadLifecycleSettingsLocked(); err != nil {
		return SaveLifecycleSettings{}, err
	}
	previous := engine.lifecycleSettings
	engine.lifecycleSettings.BackupRetention = backupRetention
	engine.lifecycleSettings.BackupNamePattern = pattern
	if err := engine.persistLifecycleSettingsLocked(); err != nil {
		engine.lifecycleSettings = previous
		return SaveLifecycleSettings{}, err
	}
	return reported(engine.lifecycleSettings), nil
}

// BackupNamePattern reports the pattern now in effect. The deployment package
// reads the setting through this method instead of holding its own copy.
func (engine *Engine) BackupNamePattern() string {
	if engine == nil {
		return backupname.Default
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadLifecycleSettingsLocked(); err != nil {
		return backupname.Default
	}
	return backupname.Normalise(engine.lifecycleSettings.BackupNamePattern)
}

// RecentFile is one host-local entry shown on Home. Path is carried exactly as
// the native host reported it and is never interpreted by the frontend.
type RecentFile struct {
	Path         string `json:"path"`
	Platform     string `json:"platform"`
	Format       string `json:"format"`
	LastOpenedAt string `json:"lastOpenedAt"`
}

type recentFilesFile struct {
	SchemaVersion int          `json:"schemaVersion"`
	Entries       []RecentFile `json:"entries"`
}

func (engine *Engine) recentFilesPath() string {
	if engine.stateDirectory == "" {
		return ""
	}
	return filepath.Join(engine.stateDirectory, "recent-files.json")
}

func (engine *Engine) loadRecentFilesLocked() error {
	if engine.recentFilesLoaded {
		return nil
	}
	path := engine.recentFilesPath()
	if path == "" {
		engine.recentFiles = []RecentFile{}
		engine.recentFilesLoaded = true
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		engine.recentFiles = []RecentFile{}
		engine.recentFilesLoaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read recent files: %w", err)
	}
	var stored recentFilesFile
	if err := json.Unmarshal(data, &stored); err != nil || stored.SchemaVersion != 1 {
		return errors.New("recent files store is invalid")
	}
	if len(stored.Entries) > maximumRecentFiles {
		stored.Entries = stored.Entries[:maximumRecentFiles]
	}
	engine.recentFiles = append([]RecentFile(nil), stored.Entries...)
	engine.recentFilesLoaded = true
	return nil
}

func (engine *Engine) persistRecentFilesLocked() error {
	path := engine.recentFilesPath()
	if path == "" {
		return nil
	}
	data, err := json.Marshal(recentFilesFile{SchemaVersion: 1, Entries: engine.recentFiles})
	if err != nil {
		return err
	}
	return writePrivateFileAtomically(path, data)
}

func (engine *Engine) GetRecentFiles() ([]RecentFile, error) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadRecentFilesLocked(); err != nil {
		return nil, err
	}
	return append([]RecentFile(nil), engine.recentFiles...), nil
}

// RecordRecentFile records only an accepted durable local session. A temporary
// deployment source is never presented as a durable recent file.
func (engine *Engine) RecordRecentFile(saveSessionID string) ([]RecentFile, error) {
	if saveSessionID == "" {
		return nil, apperror.MissingField("saveSessionID")
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return nil, apperror.UnknownSaveSession(saveSessionID)
	}
	if loaded.session.sourceKind != SourceKindLocal {
		return nil, errors.New("temporary save sessions cannot be added to Recent Files")
	}
	if err := engine.loadRecentFilesLocked(); err != nil {
		return nil, err
	}
	entry := RecentFile{
		Path:         loaded.session.sourcePath,
		Platform:     string(loaded.session.platform),
		Format:       loaded.session.format,
		LastOpenedAt: engine.nowUTC().Format(time.RFC3339Nano),
	}
	next := make([]RecentFile, 0, maximumRecentFiles)
	next = append(next, entry)
	for _, existing := range engine.recentFiles {
		if existing.Path == entry.Path {
			continue
		}
		next = append(next, existing)
		if len(next) == maximumRecentFiles {
			break
		}
	}
	previous := engine.recentFiles
	engine.recentFiles = next
	if err := engine.persistRecentFilesLocked(); err != nil {
		engine.recentFiles = previous
		return nil, err
	}
	return append([]RecentFile(nil), engine.recentFiles...), nil
}

func (engine *Engine) RemoveRecentFile(path string) ([]RecentFile, error) {
	if path == "" {
		return nil, apperror.MissingField("path")
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadRecentFilesLocked(); err != nil {
		return nil, err
	}
	next := make([]RecentFile, 0, len(engine.recentFiles))
	for _, entry := range engine.recentFiles {
		if entry.Path != path {
			next = append(next, entry)
		}
	}
	previous := engine.recentFiles
	engine.recentFiles = next
	if err := engine.persistRecentFilesLocked(); err != nil {
		engine.recentFiles = previous
		return nil, err
	}
	return append([]RecentFile(nil), engine.recentFiles...), nil
}

func (engine *Engine) ClearRecentFiles() error {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadRecentFilesLocked(); err != nil {
		return err
	}
	previous := engine.recentFiles
	engine.recentFiles = []RecentFile{}
	if err := engine.persistRecentFilesLocked(); err != nil {
		engine.recentFiles = previous
		return err
	}
	return nil
}
