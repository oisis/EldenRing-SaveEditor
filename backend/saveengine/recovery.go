package saveengine

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	recoverySchemaVersion  = 1
	sha256DigestByteLength = 32
)

type recoveryJournal struct {
	SchemaVersion     int              `json:"schemaVersion"`
	JournalID         string           `json:"journalID"`
	SourcePath        string           `json:"sourcePath"`
	SourceKind        string           `json:"sourceKind"`
	Platform          string           `json:"platform"`
	Format            string           `json:"format"`
	SourceFingerprint string           `json:"sourceFingerprint"`
	BaseRevision      string           `json:"baseRevision"`
	CurrentRevision   string           `json:"currentRevision"`
	UpdatedAt         string           `json:"updatedAt"`
	Operations        []operationEntry `json:"operations"`
}

// RecoveryJournalSummary is safe metadata used by the startup recovery flow.
// Replay bytes remain private even when the journal is corrupt or incompatible.
type RecoveryJournalSummary struct {
	JournalID      string            `json:"journalID"`
	Status         string            `json:"status"`
	SourcePath     string            `json:"sourcePath,omitempty"`
	Platform       string            `json:"platform,omitempty"`
	Format         string            `json:"format,omitempty"`
	SaveRevision   string            `json:"saveRevision,omitempty"`
	UpdatedAt      string            `json:"updatedAt,omitempty"`
	OperationCount int               `json:"operationCount"`
	Operations     []OperationRecord `json:"operations"`
	FailureCode    string            `json:"failureCode,omitempty"`
}

func (engine *Engine) recoveryDirectory() string {
	if engine.stateDirectory == "" {
		return ""
	}
	return filepath.Join(engine.stateDirectory, "recovery")
}

func (engine *Engine) recoveryPath(journalID string) (string, error) {
	if journalID == "" || filepath.Base(journalID) != journalID ||
		strings.ContainsAny(journalID, `/\\`) {
		return "", errors.New("invalid recovery journal identifier")
	}
	directory := engine.recoveryDirectory()
	if directory == "" {
		return "", nil
	}
	return filepath.Join(directory, journalID+".json"), nil
}

func (engine *Engine) persistRecoveryState(
	loaded *loadedSave,
	operations []operationEntry,
	currentRevision string,
) error {
	if engine.stateDirectory == "" {
		return nil
	}
	if loaded == nil || loaded.session == nil || loaded.baseline == nil {
		return errors.New("cannot persist recovery for an incomplete session")
	}
	path, err := engine.recoveryPath(loaded.session.id)
	if err != nil {
		return err
	}
	if len(operations) == 0 {
		return engine.removeRecoveryJournal(loaded.session.id)
	}

	journal := recoveryJournal{
		SchemaVersion:     recoverySchemaVersion,
		JournalID:         loaded.session.id,
		SourcePath:        loaded.session.sourcePath,
		SourceKind:        string(loaded.session.sourceKind),
		Platform:          string(loaded.session.platform),
		Format:            loaded.session.format,
		SourceFingerprint: loaded.sourceFingerprint,
		BaseRevision:      loaded.baselineRevision,
		CurrentRevision:   currentRevision,
		UpdatedAt:         engine.nowUTC().Format(time.RFC3339Nano),
		Operations:        make([]operationEntry, len(operations)),
	}
	for index, entry := range operations {
		journal.Operations[index] = cloneOperationEntry(entry)
	}
	encoded, err := json.Marshal(journal)
	if err != nil {
		return fmt.Errorf("cannot encode recovery journal: %w", err)
	}
	if err := writePrivateFileAtomically(path, encoded); err != nil {
		return fmt.Errorf("cannot persist recovery journal: %w", err)
	}
	return nil
}

func writePrivateFileAtomically(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".saveforge-state-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	defer temporary.Close()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	dirHandle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dirHandle.Close()
	return dirHandle.Sync()
}

func (engine *Engine) removeRecoveryJournal(journalID string) error {
	path, err := engine.recoveryPath(journalID)
	if err != nil || path == "" {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot remove recovery journal: %w", err)
	}
	return nil
}

func (engine *Engine) readRecoveryJournal(journalID string) (recoveryJournal, error) {
	path, err := engine.recoveryPath(journalID)
	if err != nil {
		return recoveryJournal{}, err
	}
	if path == "" {
		return recoveryJournal{}, errors.New("recovery persistence is not configured")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return recoveryJournal{}, err
	}
	var journal recoveryJournal
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&journal); err != nil {
		return recoveryJournal{}, fmt.Errorf("invalid recovery journal: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return recoveryJournal{}, errors.New("invalid recovery journal: trailing JSON value")
	}
	if journal.SchemaVersion != recoverySchemaVersion || journal.JournalID != journalID {
		return recoveryJournal{}, errors.New("unsupported or mismatched recovery journal")
	}
	if !isCanonicalRevision(journal.BaseRevision) ||
		!isCanonicalRevision(journal.CurrentRevision) || journal.SourcePath == "" ||
		journal.SourceFingerprint == "" || len(journal.Operations) == 0 {
		return recoveryJournal{}, errors.New("incomplete recovery journal")
	}
	baseRevision, baseErr := strconv.ParseUint(journal.BaseRevision, 10, 64)
	currentRevision, currentErr := strconv.ParseUint(journal.CurrentRevision, 10, 64)
	if baseErr != nil || currentErr != nil || baseRevision > currentRevision {
		return recoveryJournal{}, errors.New("invalid recovery revision range")
	}
	if _, err := parseSourceKind(journal.SourceKind); err != nil {
		return recoveryJournal{}, fmt.Errorf("invalid recovery source kind: %w", err)
	}
	platform, err := parseExpectedPlatform(journal.Platform)
	if err != nil || platform == "" {
		return recoveryJournal{}, errors.New("invalid recovery platform")
	}
	if (platform == PlatformPC && journal.Format != pcContainerFormat) ||
		(platform == PlatformPS4 && journal.Format != ps4ContainerFormat) {
		return recoveryJournal{}, errors.New("recovery platform and format disagree")
	}
	fingerprint, err := hex.DecodeString(journal.SourceFingerprint)
	if err != nil || len(fingerprint) != sha256DigestByteLength {
		return recoveryJournal{}, errors.New("invalid recovery source fingerprint")
	}
	for index, entry := range journal.Operations {
		if entry.Record.OperationID == "" || entry.Record.OperationKind == "" {
			return recoveryJournal{}, fmt.Errorf("recovery operation %d is incomplete", index)
		}
		if _, err := changedScopesForMutationKind(entry.Record.OperationKind); err != nil {
			return recoveryJournal{}, fmt.Errorf("recovery operation %d: %w", index, err)
		}
	}
	return journal, nil
}

func (engine *Engine) recoveryCompatibility(journal recoveryJournal) string {
	source, err := os.ReadFile(journal.SourcePath)
	if err != nil {
		return "incompatible"
	}
	if fingerprintBytes(source) != journal.SourceFingerprint {
		return "incompatible"
	}
	return "compatible"
}

func recoverySummary(journal recoveryJournal, status string) RecoveryJournalSummary {
	operations := make([]OperationRecord, len(journal.Operations))
	for index, entry := range journal.Operations {
		operations[index] = cloneOperationRecord(entry.Record)
	}
	return RecoveryJournalSummary{
		JournalID:      journal.JournalID,
		Status:         status,
		SourcePath:     journal.SourcePath,
		Platform:       journal.Platform,
		Format:         journal.Format,
		SaveRevision:   journal.CurrentRevision,
		UpdatedAt:      journal.UpdatedAt,
		OperationCount: len(journal.Operations),
		Operations:     operations,
	}
}

// GetRecoveryJournals returns every persisted journal without mutating it.
func (engine *Engine) GetRecoveryJournals() ([]RecoveryJournalSummary, error) {
	directory := engine.recoveryDirectory()
	if directory == "" {
		return []RecoveryJournalSummary{}, nil
	}
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return []RecoveryJournalSummary{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read recovery journals: %w", err)
	}
	result := make([]RecoveryJournalSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		journalID := strings.TrimSuffix(entry.Name(), ".json")
		journal, readErr := engine.readRecoveryJournal(journalID)
		if readErr != nil {
			result = append(result, RecoveryJournalSummary{
				JournalID:   journalID,
				Status:      "corrupt",
				FailureCode: "invalid_recovery_journal",
			})
			continue
		}
		result = append(result, recoverySummary(journal, engine.recoveryCompatibility(journal)))
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].UpdatedAt > result[right].UpdatedAt
	})
	return result, nil
}

func (engine *Engine) GetRecoveryJournal(journalID string) (RecoveryJournalSummary, error) {
	journal, err := engine.readRecoveryJournal(journalID)
	if err != nil {
		return RecoveryJournalSummary{}, err
	}
	return recoverySummary(journal, engine.recoveryCompatibility(journal)), nil
}

// RestoreRecoveryJournal loads the unchanged source, verifies its fingerprint,
// replays every operation in order and registers the recovered state as a new
// private session. The source file itself is never written.
func (engine *Engine) RestoreRecoveryJournal(journalID string) (SessionInfo, error) {
	journal, err := engine.readRecoveryJournal(journalID)
	if err != nil {
		return SessionInfo{}, err
	}
	if engine.recoveryCompatibility(journal) != "compatible" {
		return SessionInfo{}, errors.New("recovery journal does not match its source")
	}
	loadedInfo, err := engine.LoadSave(journal.SourcePath, journal.Platform, journal.SourceKind)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("cannot load recovery source: %w", err)
	}

	engine.mutex.Lock()
	loaded := engine.sessions[loadedInfo.SaveSessionID]
	candidate := append([]byte(nil), loaded.baseline.data...)
	for index, entry := range journal.Operations {
		if err := applyPatches(candidate, entry.Patches, true); err != nil {
			delete(engine.sessions, loadedInfo.SaveSessionID)
			engine.mutex.Unlock()
			return SessionInfo{}, fmt.Errorf("cannot replay recovery operation %d: %w", index, err)
		}
	}
	if err := validateSnapshotCandidate(loaded, candidate); err != nil {
		delete(engine.sessions, loadedInfo.SaveSessionID)
		engine.mutex.Unlock()
		return SessionInfo{}, fmt.Errorf("recovered snapshot failed validation: %w", err)
	}
	revision, err := strconv.ParseUint(journal.CurrentRevision, 10, 64)
	if err != nil {
		delete(engine.sessions, loadedInfo.SaveSessionID)
		engine.mutex.Unlock()
		return SessionInfo{}, errors.New("recovery revision is invalid")
	}
	baseRevision, err := strconv.ParseUint(journal.BaseRevision, 10, 64)
	if err != nil || baseRevision > revision {
		delete(engine.sessions, loadedInfo.SaveSessionID)
		engine.mutex.Unlock()
		return SessionInfo{}, errors.New("recovery baseline revision is invalid")
	}
	loaded.snapshot = &codec{data: candidate}
	loaded.session.revision = revision
	loaded.session.dirty = true
	loaded.baselineRevision = journal.BaseRevision
	loaded.operations = make([]operationEntry, len(journal.Operations))
	for index, entry := range journal.Operations {
		if engine.reservedOperationIDs[entry.Record.OperationID] {
			delete(engine.sessions, loadedInfo.SaveSessionID)
			engine.mutex.Unlock()
			return SessionInfo{}, fmt.Errorf("recovery operation %d reuses an operation identifier", index)
		}
	}
	for index, entry := range journal.Operations {
		cloned := cloneOperationEntry(entry)
		cloned.Record.SaveSessionID = loadedInfo.SaveSessionID
		loaded.operations[index] = cloned
		engine.reservedOperationIDs[cloned.Record.OperationID] = true
	}
	rollbackRestoredSession := func() {
		for _, entry := range loaded.operations {
			delete(engine.reservedOperationIDs, entry.Record.OperationID)
		}
		delete(engine.sessions, loadedInfo.SaveSessionID)
		_ = engine.removeRecoveryJournal(loadedInfo.SaveSessionID)
	}
	if err := engine.persistRecoveryState(loaded, loaded.operations, journal.CurrentRevision); err != nil {
		rollbackRestoredSession()
		engine.mutex.Unlock()
		return SessionInfo{}, err
	}
	if journalID != loadedInfo.SaveSessionID {
		if err := engine.removeRecoveryJournal(journalID); err != nil {
			rollbackRestoredSession()
			engine.mutex.Unlock()
			return SessionInfo{}, err
		}
	}
	result := loaded.session.Info()
	engine.mutex.Unlock()
	return result, nil
}

func (engine *Engine) DiscardRecoveryJournal(journalID string) error {
	return engine.removeRecoveryJournal(journalID)
}

// ExportRecoveryJournal copies the protected journal only to an explicit
// caller-selected destination. It never applies the journal or touches a save.
func (engine *Engine) ExportRecoveryJournal(journalID string, target string) error {
	if target == "" {
		return errors.New("export target is required")
	}
	source, err := engine.recoveryPath(journalID)
	if err != nil || source == "" {
		return err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return writeAtomically(target, data)
}
