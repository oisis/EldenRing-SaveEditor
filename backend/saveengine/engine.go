// Package saveengine owns save reading, in-memory mutation and explicit writes
// in SaveForge 2.0. Loading remains read-only; only WriteSave writes a validated
// snapshot to the caller's explicit target.
package saveengine

import (
	"fmt"
	"sync"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// magicLength is the number of leading bytes needed to recognise a container.
// Both supported magics are four bytes long.
const magicLength = 4

// Engine owns the save sessions of one backend instance. Sessions are kept under
// their own identifier because every later operation addresses one by its
// saveSessionID.
type Engine struct {
	mutex    sync.Mutex
	sessions map[string]*loadedSave
	now      func() time.Time
	// stateDirectory is the private application-data directory used by the
	// desktop lifecycle for recovery journals, recent files and lifecycle
	// settings. An empty value deliberately selects an in-memory engine; this
	// keeps package users and tests from writing host state unless the composition
	// root explicitly enables persistence.
	stateDirectory string
	// newOperationID mints the identifier of one mutation execution. A nil value
	// selects the package generator; only a test replaces it, and only to prove
	// that a generator failure or a repeated value rejects the mutation before
	// anything changes.
	newOperationID func() (string, error)
	// reservedOperationIDs holds every operationID this engine has already handed
	// to a mutation. It is read and written under mutex by mintOperationID, which
	// is what makes the identifiers of one running engine literally unique instead
	// of merely improbable to repeat.
	reservedOperationIDs map[string]bool
	// lifecycleSettings and recentFiles are host-local product state. They are
	// guarded by mutex beside the sessions and loaded lazily from stateDirectory.
	lifecycleSettings       SaveLifecycleSettings
	lifecycleSettingsLoaded bool
	// backupIndex records which local backup files this application created, so
	// retention never has to guess a file's owner from its name.
	backupIndex       map[string][]backupIndexEntry
	backupIndexLoaded bool
	recentFiles       []RecentFile
	recentFilesLoaded bool

	// eventMutex guards the session.changed outbox and its sink. It is a separate
	// lock from mutex on purpose: the outbox is drained after mutex is released,
	// so the sink never runs under the session lock. It is never held while the
	// sink runs and never taken while waiting for mutex, so the two locks cannot
	// deadlock against each other.
	eventMutex sync.Mutex
	// eventQueue holds the committed events not yet delivered. It is appended to
	// under mutex, which makes its order exactly the commit order.
	eventQueue []SessionChangedEvent
	// sessionChangedSink is the single subscriber installed by the host.
	sessionChangedSink SessionChangedSink
	// eventDrainMutex admits one drainer at a time, so queued events reach the
	// sink in order even when several mutations finish concurrently.
	eventDrainMutex sync.Mutex
}

// loadedSave is the private state of one session: its metadata model and mutable
// snapshot. Nothing here leaves the package; callers read and mutate it only
// through Engine operations.
type loadedSave struct {
	session  *Session
	snapshot *codec
	// baseline is the last successfully loaded or persisted container. Operation
	// replay, Discard Changes and recovery are all derived from this one source of
	// truth; no feature keeps a second interpretation of the save.
	baseline *codec
	// baselineRevision is the logical revision at which baseline became durable.
	// Recovery journals cannot assume that every persisted baseline is revision
	// zero because Save, Save As and Discard all advance the session revision.
	baselineRevision string
	// sourceFingerprint binds recovery and ordinary Save to the exact baseline
	// bytes from which the session was created.
	sourceFingerprint string
	// operations are the logical mutations currently applied on top of baseline.
	// redo contains entries removed by consecutive global Undo operations.
	operations []operationEntry
	redo       []operationEntry
}

// EngineOptions contains host-owned lifecycle dependencies. SaveEngine never
// guesses a writable host directory: the desktop composition root supplies it,
// while an empty value keeps persistence disabled.
type EngineOptions struct {
	StateDirectory string
}

// New returns an engine holding no session.
func New() *Engine {
	return NewWithOptions(EngineOptions{})
}

// NewWithOptions returns an engine configured by the host. Constructing it is
// side-effect free; directories and files are touched only by the lifecycle
// operation that needs them.
func NewWithOptions(options EngineOptions) *Engine {
	return &Engine{
		sessions:             make(map[string]*loadedSave),
		now:                  time.Now,
		reservedOperationIDs: make(map[string]bool),
		stateDirectory:       options.StateDirectory,
	}
}

func (engine *Engine) nowUTC() time.Time {
	if engine.now == nil {
		return time.Now().UTC()
	}
	return engine.now().UTC()
}

// LoadSave reads the local file at path read-only, recognises its platform,
// validates it against expectedPlatform, validates the container structure and
// registers a new session together with a private snapshot of the file. The
// file is never written to, it is closed before this function returns, and no
// session is created when any step fails.
//
// expectedPlatform accepts "", "pc" and "ps4". The value is matched exactly and
// case-sensitively and is never trimmed; an empty value expresses no
// expectation. Any other value is an error, and a recognised platform differing
// from a non-empty expectation is rejected before a session exists.
//
// sourceKind accepts exactly "local" and "temporary". Unlike expectedPlatform it
// has no empty form: the caller states what the file is, and an empty, aliased
// or differently cased value is an error rejected before the file system is
// touched, so no session exists for it.
//
// path is recorded on the session exactly as received. It is session metadata
// only: the snapshot is taken here and every later read goes to the snapshot, so
// the recorded path is never reopened.
func (engine *Engine) LoadSave(
	path string,
	expectedPlatform string,
	sourceKind string,
) (SessionInfo, error) {
	expected, err := parseExpectedPlatform(expectedPlatform)
	if err != nil {
		return SessionInfo{}, err
	}
	kind, err := parseSourceKind(sourceKind)
	if err != nil {
		return SessionInfo{}, err
	}

	// The codec snapshots the file and closes it before returning, so the engine
	// never holds a handle to the user's file and the session data cannot change
	// underneath it.
	source, err := openCodec(path)
	if err != nil {
		return SessionInfo{}, err
	}

	head, err := source.readAt(0, magicLength)
	if err != nil {
		return SessionInfo{}, errUnsupportedContainer
	}

	var platform Platform
	var format string
	switch {
	case pcRecognises(head):
		platform, format = PlatformPC, pcContainerFormat
	case ps4Recognises(head):
		platform, format = PlatformPS4, ps4ContainerFormat
	default:
		return SessionInfo{}, errUnsupportedContainer
	}

	if expected != "" && expected != platform {
		return SessionInfo{}, fmt.Errorf("expected a %s save, but the file is a %s save", expected, platform)
	}

	switch platform {
	case PlatformPC:
		err = pcValidate(source)
	case PlatformPS4:
		err = ps4Validate(source)
	}
	if err != nil {
		return SessionInfo{}, err
	}

	session, err := newSession(platform, format, path, kind)
	if err != nil {
		return SessionInfo{}, err
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	session.appendDiagnosticRecord(
		engine.nowUTC(),
		DiagnosticScopeSession,
		DiagnosticSeverityInfo,
		DiagnosticEventSessionLoaded,
		DiagnosticMessageSessionLoaded,
		nil,
		"0",
	)
	baseline := &codec{data: append([]byte(nil), source.data...)}
	engine.sessions[session.id] = &loadedSave{
		session:           session,
		snapshot:          source,
		baseline:          baseline,
		baselineRevision:  "0",
		sourceFingerprint: fingerprintBytes(baseline.data),
	}

	return session.Info(), nil
}

// GetSessionInfo returns the safe metadata of the session registered under
// saveSessionID. It is the only way an existing session is read back: the
// session model, its private snapshot and the save bytes never leave the
// package, and no file is opened. The source path the session records is
// metadata, so it is reported without being resolved, re-read or checked for
// existence: a source that was removed or replaced after LoadSave changes
// nothing here.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty or unknown identifier is rejected instead of resolving to a
// session. The call reads the session map and changes nothing.
func (engine *Engine) GetSessionInfo(saveSessionID string) (SessionInfo, error) {
	if saveSessionID == "" {
		return SessionInfo{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return SessionInfo{}, apperror.UnknownSaveSession(saveSessionID)
	}
	// Info returns a value, so the caller receives an independent copy of the
	// metadata and cannot reach the session behind it.
	return loaded.session.Info(), nil
}

// HasUnsavedChanges is the read-only host lifecycle check used before the
// desktop window closes. It exposes no session identifier or save data.
func (engine *Engine) HasUnsavedChanges() bool {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	for _, loaded := range engine.sessions {
		if loaded != nil && loaded.session != nil && loaded.session.dirty {
			return true
		}
	}
	return false
}

// CloseSession removes the session registered under saveSessionID. It is the
// only way a session is released: the entry is deleted from the session map, so
// the engine drops its references to the session model and to the private
// snapshot the session was created from. A clean session's stale recovery file
// is removed before the in-memory state is released; a dirty session keeps its
// recovery journal for the next startup. No save file is read or written.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty or unknown identifier is rejected instead of closing some other
// session.
//
// After any required recovery cleanup, the snapshot memory becomes eligible for
// the ordinary garbage collector; the engine never forces a collection and
// gives no timing guarantee.
func (engine *Engine) CloseSession(saveSessionID string) error {
	if saveSessionID == "" {
		return apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return apperror.UnknownSaveSession(saveSessionID)
	}
	if !loaded.session.dirty {
		if err := engine.removeRecoveryJournal(saveSessionID); err != nil {
			return err
		}
	}
	delete(engine.sessions, saveSessionID)
	return nil
}

// errUnsupportedContainer rejects every input that is not an unambiguous native
// PC or PS4 container. An encrypted or unknown container is never decrypted and
// never guessed.
var errUnsupportedContainer = fmt.Errorf(
	"unsupported save container: the file is neither a native PC nor a native PS4 save")

// parseSourceKind validates the caller's statement of what the source file is.
// It is matched exactly and case-sensitively and is never trimmed, aliased or
// defaulted: an unstated origin is a rejection, not a "local" file.
func parseSourceKind(value string) (SourceKind, error) {
	switch value {
	case string(SourceKindLocal):
		return SourceKindLocal, nil
	case string(SourceKindTemporary):
		return SourceKindTemporary, nil
	default:
		return "", fmt.Errorf("unknown source kind %q, want %q or %q",
			value, SourceKindLocal, SourceKindTemporary)
	}
}

// parseExpectedPlatform validates the caller's platform expectation.
func parseExpectedPlatform(value string) (Platform, error) {
	switch value {
	case "":
		return "", nil
	case string(PlatformPC):
		return PlatformPC, nil
	case string(PlatformPS4):
		return PlatformPS4, nil
	default:
		return "", fmt.Errorf("unknown expected platform %q, want %q, %q or an empty value",
			value, PlatformPC, PlatformPS4)
	}
}
