package saveengine

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
	"github.com/oisis/EldenRing-SaveForge/backend/backupname"
)

// SaveLifecycleResult reports a fully committed Save or Save As. Warnings are
// non-blocking post-commit maintenance failures, such as backup pruning.
type SaveLifecycleResult struct {
	MutationReceipt
	Target                  string   `json:"target"`
	BackupPath              string   `json:"backupPath,omitempty"`
	Warnings                []string `json:"warnings"`
	RetentionNoticeRequired bool     `json:"retentionNoticeRequired"`
}

func (engine *Engine) Save(
	saveSessionID string,
	expectedRevision string,
	validationToken string,
	confirmWarnings bool,
	confirmBanRisk bool,
) (SaveLifecycleResult, error) {
	return engine.saveLifecycle(
		saveSessionID, expectedRevision, validationToken,
		confirmWarnings, confirmBanRisk, "", false)
}

func (engine *Engine) SaveAs(
	saveSessionID string,
	expectedRevision string,
	validationToken string,
	confirmWarnings bool,
	confirmBanRisk bool,
	target string,
) (SaveLifecycleResult, error) {
	if target == "" {
		return SaveLifecycleResult{}, apperror.MissingField("target")
	}
	return engine.saveLifecycle(
		saveSessionID, expectedRevision, validationToken,
		confirmWarnings, confirmBanRisk, target, true)
}

func (engine *Engine) saveLifecycle(
	saveSessionID string,
	expectedRevision string,
	validationToken string,
	confirmWarnings bool,
	confirmBanRisk bool,
	target string,
	saveAs bool,
) (SaveLifecycleResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SaveLifecycleResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if saveSessionID == "" {
		return SaveLifecycleResult{}, apperror.MissingField("saveSessionID")
	}

	defer engine.publishSessionChanged()
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return SaveLifecycleResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	session := loaded.session
	if expectedRevision != session.revisionString() {
		return SaveLifecycleResult{}, apperror.RevisionConflict(
			expectedRevision, session.revisionString())
	}
	if err := requireReviewAuthorization(
		session, expectedRevision, validationToken, confirmWarnings, confirmBanRisk); err != nil {
		return SaveLifecycleResult{}, err
	}
	operationKind := kindSave
	if saveAs {
		operationKind = kindSaveAs
	} else {
		if session.sourceKind != SourceKindLocal || session.sourcePath == "" {
			return SaveLifecycleResult{}, errors.New("Save requires a durable local target; use Save As")
		}
		target = session.sourcePath
	}

	pending, err := engine.prepareMutation(operationKind)
	if err != nil {
		return SaveLifecycleResult{}, err
	}
	if err := engine.loadLifecycleSettingsLocked(); err != nil {
		return SaveLifecycleResult{}, err
	}

	backupPath := ""
	targetExists, err := regularFileExists(target)
	if err != nil {
		return SaveLifecycleResult{}, err
	}
	expectedTargetFingerprint := ""
	if targetExists {
		expectedTargetFingerprint, err = fingerprintFile(target)
		if err != nil {
			return SaveLifecycleResult{}, err
		}
	}
	if !saveAs && expectedTargetFingerprint != loaded.sourceFingerprint {
		return SaveLifecycleResult{}, errors.New(
			"the save target changed outside SaveForge; use Save As or reload it")
	}
	if targetExists {
		backupTime := engine.nowUTC()
		backupPath, err = createAutomaticBackup(
			target, backupTime, backupname.Normalise(engine.lifecycleSettings.BackupNamePattern))
		if err != nil {
			return SaveLifecycleResult{}, fmt.Errorf("cannot create required backup: %w", err)
		}
		// The backup is recorded before the target is touched, so retention can
		// always tell this application's file from somebody else's.
		if err := engine.recordAutomaticBackupLocked(
			target, filepath.Base(backupPath), backupTime); err != nil {
			return SaveLifecycleResult{}, fmt.Errorf("cannot record required backup: %w", err)
		}
		if err := verifyTargetFingerprint(backupPath, expectedTargetFingerprint); err != nil {
			return SaveLifecycleResult{}, fmt.Errorf(
				"automatic backup does not match the target before replacement: %w", err)
		}
	}

	candidate, err := serializeContainer(loaded)
	if err != nil {
		return SaveLifecycleResult{}, fmt.Errorf("cannot serialize save session: %w", err)
	}
	if err := validateSerialized(candidate, session.platform); err != nil {
		return SaveLifecycleResult{}, fmt.Errorf("cannot validate serialized save: %w", err)
	}
	if targetExists {
		if err := verifyTargetFingerprint(target, expectedTargetFingerprint); err != nil {
			return SaveLifecycleResult{}, err
		}
	}
	if err := writeAtomically(target, candidate); err != nil {
		return SaveLifecycleResult{}, fmt.Errorf("cannot replace save target: %w", err)
	}
	if err := verifyWrittenTarget(target, candidate, session.platform); err != nil {
		rollbackErr := rollbackCommittedTarget(target, backupPath)
		if rollbackErr != nil {
			return SaveLifecycleResult{}, fmt.Errorf(
				"written target failed final validation: %v; rollback also failed: %w", err, rollbackErr)
		}
		return SaveLifecycleResult{}, fmt.Errorf("written target failed final validation and was rolled back: %w", err)
	}

	warnings := []string{}
	retentionNotice := false
	if targetExists {
		owned, ownedErr := engine.ownedAutomaticBackupsLocked(target)
		removedNames, reachedLimit, pruneErr := []string(nil), false, ownedErr
		if pruneErr == nil {
			removedNames, reachedLimit, pruneErr = pruneAutomaticBackups(
				target, engine.lifecycleSettings.BackupRetention, owned)
		}
		if pruneErr == nil {
			pruneErr = engine.forgetAutomaticBackupsLocked(target, removedNames)
		}
		if pruneErr != nil {
			warnings = append(warnings, "The save was written, but old automatic backups could not be pruned.")
		} else if reachedLimit && !engine.lifecycleSettings.RetentionNoticeShown {
			retentionNotice = true
			engine.lifecycleSettings.RetentionNoticeShown = true
			if settingsErr := engine.persistLifecycleSettingsLocked(); settingsErr != nil {
				warnings = append(warnings, "The save was written, but the backup retention notice could not be remembered.")
			}
		}
	}
	if err := engine.removeRecoveryJournal(saveSessionID); err != nil {
		warnings = append(warnings, "The save was written, but its recovery journal could not be removed.")
	}

	loaded.snapshot = &codec{data: append([]byte(nil), candidate...)}
	loaded.baseline = &codec{data: append([]byte(nil), candidate...)}
	loaded.sourceFingerprint = fingerprintBytes(candidate)
	loaded.operations = nil
	loaded.redo = nil
	session.sourcePath = target
	session.sourceKind = SourceKindLocal
	session.dirty = false
	session.undo = nil
	session.reviewAuthorization = nil
	receipt := pending.receipt(saveSessionID, session.advanceRevision())
	loaded.baselineRevision = receipt.SaveRevision
	session.appendDiagnosticRecord(
		engine.nowUTC(),
		DiagnosticScopeSession,
		DiagnosticSeverityInfo,
		DiagnosticEventSaveWritten,
		DiagnosticMessageSaveWritten,
		nil,
		receipt.SaveRevision,
	)
	engine.enqueueCommitted(session, receipt)
	return SaveLifecycleResult{
		MutationReceipt:         receipt,
		Target:                  target,
		BackupPath:              backupPath,
		Warnings:                warnings,
		RetentionNoticeRequired: retentionNotice,
	}, nil
}

func verifyTargetFingerprint(target string, expected string) error {
	current, err := fingerprintFile(target)
	if err != nil {
		return err
	}
	if current != expected {
		return errors.New("the save target changed outside SaveForge; use Save As or reload it")
	}
	return nil
}

func fingerprintFile(target string) (string, error) {
	current, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("cannot verify current save target: %w", err)
	}
	return fingerprintBytes(current), nil
}

func regularFileExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("save target %q is not a regular file", path)
		}
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("cannot inspect save target %q: %w", path, err)
	}
}

// createAutomaticBackup writes the backup beside the target under the name the
// configured pattern renders. The pattern grammar, the suffix and the collision
// counter all live in one package, so a local backup and a deployment target
// backup are named by the same rules.
func createAutomaticBackup(target string, now time.Time, pattern string) (string, error) {
	source, err := os.Open(target)
	if err != nil {
		return "", err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("backup source is not a regular file")
	}

	directory := filepath.Dir(target)
	sourceName := filepath.Base(target)
	for collision := 1; collision <= 10000; collision++ {
		name, nameErr := backupname.Candidate(pattern, sourceName, now, collision)
		if nameErr != nil {
			return "", nameErr
		}
		path := filepath.Join(directory, name)
		destination, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if errors.Is(openErr, os.ErrExist) {
			continue
		}
		if openErr != nil {
			return "", openErr
		}
		copyErr := func() error {
			if _, err := io.Copy(destination, source); err != nil {
				_ = destination.Close()
				return err
			}
			if err := destination.Sync(); err != nil {
				_ = destination.Close()
				return err
			}
			return destination.Close()
		}()
		if copyErr != nil {
			_ = os.Remove(path)
			return "", copyErr
		}
		return path, nil
	}
	return "", errors.New("cannot allocate a unique automatic backup name")
}

type automaticBackup struct {
	name    string
	path    string
	modTime time.Time
}

// pruneAutomaticBackups removes the automatic backups of one save past the
// retention limit and reports the file names it removed.
//
// owned maps the file name of every backup this application recorded for the
// save onto its recorded creation time; it may be nil. A file is a candidate
// only when it is recorded there or when its name is one the fixed 2.0 grammar
// produced. A file that is neither belongs to somebody else and is never
// touched, whatever pattern is configured now.
func pruneAutomaticBackups(
	target string, retention int, owned map[string]time.Time,
) ([]string, bool, error) {
	directory := filepath.Dir(target)
	sourceName := filepath.Base(target)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, false, err
	}
	backups := make([]automaticBackup, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		recorded, isOwned := owned[entry.Name()]
		if !isOwned {
			if _, matches := backupname.MatchesDefault(entry.Name(), sourceName); !matches {
				continue
			}
		}
		info, err := entry.Info()
		if err != nil {
			return nil, false, err
		}
		// The recorded creation time is authoritative when this application wrote
		// the file; a file only recognised by its name keeps the modification time
		// it has always been ordered by.
		when := info.ModTime()
		if isOwned {
			when = recorded
		}
		backups = append(backups, automaticBackup{
			name:    entry.Name(),
			path:    filepath.Join(directory, entry.Name()),
			modTime: when,
		})
	}
	if len(backups) <= retention {
		return nil, len(backups) == retention, nil
	}
	sort.SliceStable(backups, func(left, right int) bool {
		if backups[left].modTime.Equal(backups[right].modTime) {
			return backups[left].path < backups[right].path
		}
		return backups[left].modTime.Before(backups[right].modTime)
	})
	removed := make([]string, 0, len(backups)-retention)
	for _, backup := range backups[:len(backups)-retention] {
		if err := os.Remove(backup.path); err != nil {
			return nil, false, err
		}
		removed = append(removed, backup.name)
	}
	return removed, true, nil
}

func verifyWrittenTarget(target string, candidate []byte, platform Platform) error {
	written, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	if !bytes.Equal(written, candidate) {
		return errors.New("written file differs from the validated candidate")
	}
	return validateSerialized(written, platform)
}

func rollbackCommittedTarget(target string, backupPath string) error {
	if backupPath == "" {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return writeAtomically(target, backup)
}
