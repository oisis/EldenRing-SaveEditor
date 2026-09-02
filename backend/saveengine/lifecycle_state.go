package saveengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

const (
	defaultBackupRetention = 10
	maximumRecentFiles     = 10
)

// SaveLifecycleSettings contains the agreed configurable retention policy.
// Backup naming remains the fixed canonical product pattern until a public
// custom-pattern grammar is specified.
type SaveLifecycleSettings struct {
	BackupRetention      int  `json:"backupRetention"`
	RetentionNoticeShown bool `json:"retentionNoticeShown"`
}

type lifecycleSettingsFile struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Settings      SaveLifecycleSettings `json:"settings"`
}

func defaultSaveLifecycleSettings() SaveLifecycleSettings {
	return SaveLifecycleSettings{BackupRetention: defaultBackupRetention}
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
	return engine.lifecycleSettings, nil
}

func (engine *Engine) SetSaveLifecycleSettings(
	backupRetention int,
) (SaveLifecycleSettings, error) {
	if err := validateBackupRetention(backupRetention); err != nil {
		return SaveLifecycleSettings{}, err
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.loadLifecycleSettingsLocked(); err != nil {
		return SaveLifecycleSettings{}, err
	}
	previous := engine.lifecycleSettings
	engine.lifecycleSettings.BackupRetention = backupRetention
	if err := engine.persistLifecycleSettingsLocked(); err != nil {
		engine.lifecycleSettings = previous
		return SaveLifecycleSettings{}, err
	}
	return engine.lifecycleSettings, nil
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
