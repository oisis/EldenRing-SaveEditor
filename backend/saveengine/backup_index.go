package saveengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// The provenance index of the local automatic backups.
//
// A backup written under the default name states its owner and its creation
// time in the name itself. A backup written under a configured pattern does
// not, and retention may not guess: deleting a file because it happens to sit
// beside the save and end in the backup suffix would let a configuration change
// remove somebody else's file.
//
// So every backup this application creates is recorded here with the file it
// protects and the moment it was taken. Retention considers exactly two things:
// what this index records, and the names the fixed 2.0 grammar produced before
// the index existed. Nothing else is ever a candidate.

const backupIndexSchemaVersion = 1

type backupIndexEntry struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type backupIndexFile struct {
	SchemaVersion int                           `json:"schemaVersion"`
	Sources       map[string][]backupIndexEntry `json:"sources"`
}

func (engine *Engine) backupIndexPath() string {
	if engine.stateDirectory == "" {
		return ""
	}
	return filepath.Join(engine.stateDirectory, "local-backups.json")
}

func (engine *Engine) loadBackupIndexLocked() error {
	if engine.backupIndexLoaded {
		return nil
	}
	engine.backupIndex = map[string][]backupIndexEntry{}
	path := engine.backupIndexPath()
	if path == "" {
		engine.backupIndexLoaded = true
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		engine.backupIndexLoaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read the local backup index: %w", err)
	}
	var stored backupIndexFile
	if err := json.Unmarshal(data, &stored); err != nil ||
		stored.SchemaVersion != backupIndexSchemaVersion {
		// Fail closed. An index that cannot be read must not become an empty one:
		// that would hide every custom-named backup from retention rather than
		// deleting the wrong file, but it would also silently lose the record.
		return errors.New("the local backup index is invalid")
	}
	if stored.Sources != nil {
		engine.backupIndex = stored.Sources
	}
	engine.backupIndexLoaded = true
	return nil
}

func (engine *Engine) persistBackupIndexLocked() error {
	path := engine.backupIndexPath()
	if path == "" {
		return nil
	}
	data, err := json.Marshal(backupIndexFile{
		SchemaVersion: backupIndexSchemaVersion,
		Sources:       engine.backupIndex,
	})
	if err != nil {
		return err
	}
	return writePrivateFileAtomically(path, data)
}

// recordAutomaticBackupLocked notes one backup this application created.
func (engine *Engine) recordAutomaticBackupLocked(target string, name string, when time.Time) error {
	if err := engine.loadBackupIndexLocked(); err != nil {
		return err
	}
	engine.backupIndex[target] = append(
		engine.backupIndex[target], backupIndexEntry{Name: name, CreatedAt: when.UTC()})
	return engine.persistBackupIndexLocked()
}

// ownedAutomaticBackupsLocked reports the recorded creation time of every
// backup this application created for one save, keyed by file name.
func (engine *Engine) ownedAutomaticBackupsLocked(target string) (map[string]time.Time, error) {
	if err := engine.loadBackupIndexLocked(); err != nil {
		return nil, err
	}
	owned := make(map[string]time.Time, len(engine.backupIndex[target]))
	for _, entry := range engine.backupIndex[target] {
		owned[entry.Name] = entry.CreatedAt
	}
	return owned, nil
}

// forgetAutomaticBackupsLocked drops the records of files retention removed.
func (engine *Engine) forgetAutomaticBackupsLocked(target string, removed []string) error {
	if len(removed) == 0 {
		return nil
	}
	if err := engine.loadBackupIndexLocked(); err != nil {
		return err
	}
	gone := make(map[string]bool, len(removed))
	for _, name := range removed {
		gone[name] = true
	}
	kept := make([]backupIndexEntry, 0, len(engine.backupIndex[target]))
	for _, entry := range engine.backupIndex[target] {
		if !gone[entry.Name] {
			kept = append(kept, entry)
		}
	}
	if len(kept) == 0 {
		delete(engine.backupIndex, target)
	} else {
		engine.backupIndex[target] = kept
	}
	return engine.persistBackupIndexLocked()
}
