package deployment

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const storeSchemaVersion = 1

// BackupRecord is the host-owned metadata of one backup file that lives on a
// target. The file itself is on the target; this record is what Save Manager
// presents and what makes a backup manual, tagged, described or active.
//
// There is deliberately no size field: section 4.10.4 of frontend.md states the
// table has no Size column, and a value nothing renders would only be one more
// thing that can go stale.
type BackupRecord struct {
	ID        string `json:"id"`
	TargetID  string `json:"targetID"`
	FileName  string `json:"fileName"`
	CreatedAt string `json:"createdAt"`
	// Manual marks a backup the user asked for. Section 11 of deployment.md
	// exempts those from automatic retention, so retention never sees them.
	Manual bool `json:"manual"`
	// Active marks the backup the user declared as the target's active save. At
	// most one record per target carries it.
	Active      bool     `json:"active"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
}

type storeFile struct {
	SchemaVersion   int                       `json:"schemaVersion"`
	Targets         []Target                  `json:"targets"`
	TrustedHostKeys map[string]string         `json:"trustedHostKeys,omitempty"`
	Backups         map[string][]BackupRecord `json:"backups,omitempty"`
}

// Store persists the deployment configuration in the host state directory
// supplied by the composition root. An empty directory selects an in-memory
// mode: nothing is written, which is what keeps the endpoints exercisable
// without a host.
type Store struct {
	mutex     sync.Mutex
	directory string
	state     storeFile
	loaded    bool
}

// NewStore returns a store backed by directory.
func NewStore(directory string) *Store {
	return &Store{directory: directory}
}

func (store *Store) path() string {
	if store.directory == "" {
		return ""
	}
	return filepath.Join(store.directory, "deployment.json")
}

func newIdentifier(prefix string) (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cannot create an identifier: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(raw), nil
}

// ListTargets reports every configured target, ordered by name and tie-broken
// by identifier so two targets that share a name keep a stable order.
func (store *Store) ListTargets() ([]Target, error) {
	if store == nil {
		return nil, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return nil, err
	}
	targets := append([]Target(nil), store.state.Targets...)
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].Name != targets[j].Name {
			return targets[i].Name < targets[j].Name
		}
		return targets[i].ID < targets[j].ID
	})
	return targets, nil
}

// GetTarget reports one target by identifier.
func (store *Store) GetTarget(targetID string) (Target, error) {
	if store == nil {
		return Target{}, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return Target{}, err
	}
	index := store.indexOfLocked(targetID)
	if index < 0 {
		return Target{}, fmt.Errorf("unknown deployment target %q", targetID)
	}
	return store.state.Targets[index], nil
}

// CreateTarget validates and stores a new target and returns it with the
// identifier the store assigned. A caller-supplied identifier is ignored: the
// store owns identity, so a frontend can never overwrite one target by claiming
// another one's identifier on a create.
func (store *Store) CreateTarget(target Target) (Target, error) {
	if store == nil {
		return Target{}, errors.New("deployment store is not available")
	}
	if err := target.Validate(); err != nil {
		return Target{}, err
	}
	id, err := newIdentifier("target")
	if err != nil {
		return Target{}, err
	}
	target.ID = id
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return Target{}, err
	}
	previous := store.state.Targets
	store.state.Targets = append(append([]Target(nil), previous...), target)
	if err := store.persistLocked(); err != nil {
		store.state.Targets = previous
		return Target{}, err
	}
	return target, nil
}

// UpdateTarget replaces one stored target. Changing the host or the port of an
// SSH target drops the fingerprint trusted for its previous address, so a
// reconfigured target is never silently trusted on the strength of a decision
// the user made about a different machine.
func (store *Store) UpdateTarget(target Target) (Target, error) {
	if store == nil {
		return Target{}, errors.New("deployment store is not available")
	}
	if target.ID == "" {
		return Target{}, errors.New("a deployment target update needs the target identifier")
	}
	if err := target.Validate(); err != nil {
		return Target{}, err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return Target{}, err
	}
	index := store.indexOfLocked(target.ID)
	if index < 0 {
		return Target{}, fmt.Errorf("unknown deployment target %q", target.ID)
	}
	previousTarget := store.state.Targets[index]
	previousKeys := cloneStringMap(store.state.TrustedHostKeys)
	store.state.Targets[index] = target
	if previousTarget.Kind == KindSSH &&
		(target.Kind != KindSSH || previousTarget.Address() != target.Address()) {
		delete(store.state.TrustedHostKeys, previousTarget.Address())
	}
	if err := store.persistLocked(); err != nil {
		store.state.Targets[index] = previousTarget
		store.state.TrustedHostKeys = previousKeys
		return Target{}, err
	}
	return target, nil
}

// DeleteTarget removes one target together with its trusted host key and its
// backup metadata. Nothing on the target itself is touched: removing a
// configuration entry may never delete a user's files.
func (store *Store) DeleteTarget(targetID string) error {
	if store == nil {
		return errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return err
	}
	index := store.indexOfLocked(targetID)
	if index < 0 {
		return fmt.Errorf("unknown deployment target %q", targetID)
	}
	removed := store.state.Targets[index]
	previousTargets := append([]Target(nil), store.state.Targets...)
	previousKeys := cloneStringMap(store.state.TrustedHostKeys)
	previousBackups := store.state.Backups[targetID]

	store.state.Targets = append(store.state.Targets[:index:index], store.state.Targets[index+1:]...)
	if removed.Kind == KindSSH {
		delete(store.state.TrustedHostKeys, removed.Address())
	}
	delete(store.state.Backups, targetID)
	if err := store.persistLocked(); err != nil {
		store.state.Targets = previousTargets
		store.state.TrustedHostKeys = previousKeys
		if previousBackups != nil {
			store.ensureBackupsLocked()
			store.state.Backups[targetID] = previousBackups
		}
		return err
	}
	return nil
}

// TrustedHostKey reports the fingerprint remembered for an address.
func (store *Store) TrustedHostKey(address string) (string, bool, error) {
	if store == nil {
		return "", false, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return "", false, err
	}
	fingerprint, known := store.state.TrustedHostKeys[address]
	return fingerprint, known, nil
}

// TrustHostKey remembers one fingerprint for one address. It is only ever
// called from an explicit user approval: nothing in the connection path trusts
// a key on its own, which is what makes the trust decision Trust On First Use
// rather than trust on every use.
func (store *Store) TrustHostKey(address string, fingerprint string) error {
	if store == nil {
		return errors.New("deployment store is not available")
	}
	if strings.TrimSpace(address) == "" || strings.TrimSpace(fingerprint) == "" {
		return errors.New("trusting a host key needs the address and the fingerprint")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return err
	}
	previous := cloneStringMap(store.state.TrustedHostKeys)
	if store.state.TrustedHostKeys == nil {
		store.state.TrustedHostKeys = map[string]string{}
	}
	store.state.TrustedHostKeys[address] = fingerprint
	if err := store.persistLocked(); err != nil {
		store.state.TrustedHostKeys = previous
		return err
	}
	return nil
}

// ForgetHostKey drops the fingerprint remembered for an address, so the next
// connection asks for approval again.
func (store *Store) ForgetHostKey(address string) error {
	if store == nil {
		return errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return err
	}
	previous := cloneStringMap(store.state.TrustedHostKeys)
	delete(store.state.TrustedHostKeys, address)
	if err := store.persistLocked(); err != nil {
		store.state.TrustedHostKeys = previous
		return err
	}
	return nil
}

// ListBackups reports the backup metadata of one target, newest first.
func (store *Store) ListBackups(targetID string) ([]BackupRecord, error) {
	if store == nil {
		return nil, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return nil, err
	}
	records := append([]BackupRecord(nil), store.state.Backups[targetID]...)
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CreatedAt != records[j].CreatedAt {
			return records[i].CreatedAt > records[j].CreatedAt
		}
		return records[i].ID < records[j].ID
	})
	return records, nil
}

// AddBackup stores the metadata of a backup file that already exists on the
// target. The caller creates the file first and records it here afterwards, so
// a record can never describe a backup that was never written.
func (store *Store) AddBackup(record BackupRecord) (BackupRecord, error) {
	if store == nil {
		return BackupRecord{}, errors.New("deployment store is not available")
	}
	if record.TargetID == "" || record.FileName == "" {
		return BackupRecord{}, errors.New("a backup record needs its target and file name")
	}
	id, err := newIdentifier("backup")
	if err != nil {
		return BackupRecord{}, err
	}
	record.ID = id
	record.Active = false
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return BackupRecord{}, err
	}
	previous := append([]BackupRecord(nil), store.state.Backups[record.TargetID]...)
	store.ensureBackupsLocked()
	store.state.Backups[record.TargetID] = append(store.state.Backups[record.TargetID], record)
	if err := store.persistLocked(); err != nil {
		store.state.Backups[record.TargetID] = previous
		return BackupRecord{}, err
	}
	return record, nil
}

// UpdateBackupMetadata replaces the tags and the description of one backup.
func (store *Store) UpdateBackupMetadata(
	targetID string, backupID string, tags []string, description string,
) (BackupRecord, error) {
	return store.mutateBackup(targetID, backupID, func(records []BackupRecord, index int) error {
		records[index].Tags = append([]string(nil), tags...)
		records[index].Description = description
		return nil
	})
}

// SetActiveBackup marks one backup as the target's active save, or clears the
// mark when backupID is empty. An unknown identifier is refused, so the user
// can never mark a backup that does not exist.
func (store *Store) SetActiveBackup(targetID string, backupID string) ([]BackupRecord, error) {
	if store == nil {
		return nil, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return nil, err
	}
	records := store.state.Backups[targetID]
	if backupID != "" && indexOfBackup(records, backupID) < 0 {
		return nil, fmt.Errorf("unknown backup %q", backupID)
	}
	previous := append([]BackupRecord(nil), records...)
	updated := append([]BackupRecord(nil), records...)
	for index := range updated {
		updated[index].Active = backupID != "" && updated[index].ID == backupID
	}
	store.ensureBackupsLocked()
	store.state.Backups[targetID] = updated
	if err := store.persistLocked(); err != nil {
		store.state.Backups[targetID] = previous
		return nil, err
	}
	return append([]BackupRecord(nil), updated...), nil
}

// RemoveBackup drops one backup record and reports what it described, so the
// caller can delete the matching file on the target.
func (store *Store) RemoveBackup(targetID string, backupID string) (BackupRecord, error) {
	if store == nil {
		return BackupRecord{}, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return BackupRecord{}, err
	}
	records := store.state.Backups[targetID]
	index := indexOfBackup(records, backupID)
	if index < 0 {
		return BackupRecord{}, fmt.Errorf("unknown backup %q", backupID)
	}
	removed := records[index]
	previous := append([]BackupRecord(nil), records...)
	store.ensureBackupsLocked()
	store.state.Backups[targetID] = append(records[:index:index], records[index+1:]...)
	if err := store.persistLocked(); err != nil {
		store.state.Backups[targetID] = previous
		return BackupRecord{}, err
	}
	return removed, nil
}

// AutomaticBackupsOverRetention reports the automatic backups of a target that
// exceed the retention limit, oldest first. Manual backups are never returned:
// section 11 of deployment.md exempts them from automatic removal, so the
// pruning path cannot even see them.
func (store *Store) AutomaticBackupsOverRetention(
	targetID string, retention int,
) ([]BackupRecord, error) {
	if retention < 1 {
		return nil, nil
	}
	records, err := store.ListBackups(targetID)
	if err != nil {
		return nil, err
	}
	automatic := make([]BackupRecord, 0, len(records))
	for _, record := range records {
		if !record.Manual && !record.Active {
			automatic = append(automatic, record)
		}
	}
	if len(automatic) <= retention {
		return nil, nil
	}
	// ListBackups is newest first, so everything past the retention window is
	// the tail of that order.
	return append([]BackupRecord(nil), automatic[retention:]...), nil
}

func (store *Store) mutateBackup(
	targetID string, backupID string, apply func([]BackupRecord, int) error,
) (BackupRecord, error) {
	if store == nil {
		return BackupRecord{}, errors.New("deployment store is not available")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.loadLocked(); err != nil {
		return BackupRecord{}, err
	}
	records := store.state.Backups[targetID]
	index := indexOfBackup(records, backupID)
	if index < 0 {
		return BackupRecord{}, fmt.Errorf("unknown backup %q", backupID)
	}
	previous := append([]BackupRecord(nil), records...)
	updated := append([]BackupRecord(nil), records...)
	if err := apply(updated, index); err != nil {
		return BackupRecord{}, err
	}
	store.ensureBackupsLocked()
	store.state.Backups[targetID] = updated
	if err := store.persistLocked(); err != nil {
		store.state.Backups[targetID] = previous
		return BackupRecord{}, err
	}
	return updated[index], nil
}

func indexOfBackup(records []BackupRecord, backupID string) int {
	for index, record := range records {
		if record.ID == backupID {
			return index
		}
	}
	return -1
}

func (store *Store) indexOfLocked(targetID string) int {
	for index, target := range store.state.Targets {
		if target.ID == targetID {
			return index
		}
	}
	return -1
}

func (store *Store) ensureBackupsLocked() {
	if store.state.Backups == nil {
		store.state.Backups = map[string][]BackupRecord{}
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func (store *Store) loadLocked() error {
	if store.loaded {
		return nil
	}
	store.state = storeFile{
		SchemaVersion:   storeSchemaVersion,
		TrustedHostKeys: map[string]string{},
		Backups:         map[string][]BackupRecord{},
	}
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
		return fmt.Errorf("cannot read the deployment configuration: %w", err)
	}
	var stored storeFile
	if err := json.Unmarshal(data, &stored); err != nil ||
		stored.SchemaVersion != storeSchemaVersion {
		// Fail closed: a configuration that cannot be understood must not become
		// an empty one, because that would silently drop every target and every
		// trusted host key the user relies on.
		return errors.New("the deployment configuration is invalid")
	}
	if stored.TrustedHostKeys == nil {
		stored.TrustedHostKeys = map[string]string{}
	}
	if stored.Backups == nil {
		stored.Backups = map[string][]BackupRecord{}
	}
	store.state = stored
	store.loaded = true
	return nil
}

func (store *Store) persistLocked() error {
	path := store.path()
	if path == "" {
		return nil
	}
	store.state.SchemaVersion = storeSchemaVersion
	data, err := json.Marshal(store.state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create the deployment configuration directory: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("cannot write the deployment configuration: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("cannot store the deployment configuration: %w", err)
	}
	return nil
}
