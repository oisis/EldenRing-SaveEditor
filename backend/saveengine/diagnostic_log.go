package saveengine

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Allowed diagnostic severity levels.
const (
	DiagnosticSeverityInfo    = "info"
	DiagnosticSeverityWarning = "warning"
	DiagnosticSeverityError   = "error"
)

// Allowed diagnostic scopes in v1.
const (
	DiagnosticScopeSession = "session"
	DiagnosticScopeRepairs = "repairs"
)

// Fixed diagnostic events in v1.
const (
	DiagnosticEventSessionLoaded  = "session_loaded"
	DiagnosticEventSaveWritten    = "save_written"
	DiagnosticEventRepairsApplied = "repairs_applied"
)

// Fixed diagnostic messages in v1.
const (
	DiagnosticMessageSessionLoaded  = "save session loaded and validated"
	DiagnosticMessageSaveWritten    = "save snapshot written and verified"
	DiagnosticMessageRepairsApplied = "repair plan actions executed"
)

// journalCapacity is the fixed capacity of the per-session ring buffer.
const journalCapacity = 500

// DiagnosticRecord represents a single safe, structured diagnostic entry.
type DiagnosticRecord struct {
	Seq         uint64 `json:"seq"`
	Timestamp   string `json:"timestamp"`
	Severity    string `json:"severity"`
	Scope       string `json:"scope"`
	Event       string `json:"event"`
	Message     string `json:"message"`
	CharacterID *int   `json:"characterID,omitempty"`
	Revision    string `json:"revision,omitempty"`
}

// DiagnosticLogResult carries a page of diagnostic records and pagination state.
type DiagnosticLogResult struct {
	SaveSessionID         string             `json:"saveSessionID"`
	Records               []DiagnosticRecord `json:"records"`
	NextCursor            string             `json:"nextCursor"`
	HasMore               bool               `json:"hasMore"`
	TotalBuffered         int                `json:"totalBuffered"`
	CursorExpired         bool               `json:"cursorExpired"`
	OldestAvailableCursor string             `json:"oldestAvailableCursor"`
}

// appendDiagnosticRecord appends a new record to the session's ring buffer under the engine lock.
func (session *Session) appendDiagnosticRecord(
	now time.Time,
	scope string,
	severity string,
	event string,
	message string,
	characterID *int,
	revision string,
) {
	session.journalSeq++
	var charIDCopy *int
	if characterID != nil {
		val := *characterID
		charIDCopy = &val
	}
	record := DiagnosticRecord{
		Seq:         session.journalSeq,
		Timestamp:   now.UTC().Format(time.RFC3339),
		Severity:    severity,
		Scope:       scope,
		Event:       event,
		Message:     message,
		CharacterID: charIDCopy,
		Revision:    revision,
	}

	if len(session.journal) < journalCapacity {
		session.journal = append(session.journal, record)
		return
	}

	copy(session.journal, session.journal[1:])
	session.journal[journalCapacity-1] = record
}

// GetDiagnosticLog returns a safe, filtered page of the session's diagnostic journal.
func (engine *Engine) GetDiagnosticLog(
	saveSessionID string,
	cursor string,
	limit int,
	severity string,
	scope string,
) (DiagnosticLogResult, error) {
	if saveSessionID == "" {
		return DiagnosticLogResult{}, errors.New("saveSessionID is required")
	}

	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 200 {
		return DiagnosticLogResult{}, fmt.Errorf("limit %d is outside the range 1..200", limit)
	}

	switch severity {
	case "", DiagnosticSeverityInfo, DiagnosticSeverityWarning, DiagnosticSeverityError:
	default:
		return DiagnosticLogResult{}, fmt.Errorf(
			"unknown severity %q, want %q, %q, %q or an empty value",
			severity, DiagnosticSeverityInfo, DiagnosticSeverityWarning, DiagnosticSeverityError,
		)
	}

	switch scope {
	case "", DiagnosticScopeSession, DiagnosticScopeRepairs:
	default:
		return DiagnosticLogResult{}, fmt.Errorf(
			"unknown scope %q, want %q, %q or an empty value",
			scope, DiagnosticScopeSession, DiagnosticScopeRepairs,
		)
	}

	var cursorSeq uint64
	var hasCursor bool
	if cursor != "" {
		parsed, err := strconv.ParseUint(cursor, 10, 64)
		if err != nil || strconv.FormatUint(parsed, 10) != cursor {
			return DiagnosticLogResult{}, fmt.Errorf(
				"cursor must be a canonical decimal uint64; got %q", cursor,
			)
		}
		cursorSeq = parsed
		hasCursor = true
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()

	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return DiagnosticLogResult{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}

	session := loaded.session
	totalBuffered := len(session.journal)
	if totalBuffered == 0 {
		return DiagnosticLogResult{
			SaveSessionID:         saveSessionID,
			Records:               []DiagnosticRecord{},
			NextCursor:            "",
			HasMore:               false,
			TotalBuffered:         0,
			CursorExpired:         false,
			OldestAvailableCursor: "",
		}, nil
	}

	oldestSeq := session.journal[0].Seq
	newestSeq := session.journal[totalBuffered-1].Seq
	oldestCursorStr := strconv.FormatUint(oldestSeq, 10)

	cursorExpired := false
	if hasCursor {
		if cursorSeq < oldestSeq-1 {
			cursorExpired = true
		}
	}

	// Filter matching records
	var matched []DiagnosticRecord
	for _, rec := range session.journal {
		if !cursorExpired && hasCursor {
			if rec.Seq <= cursorSeq {
				continue
			}
		}
		if severity != "" && rec.Severity != severity {
			continue
		}
		if scope != "" && rec.Scope != scope {
			continue
		}
		// Deep copy record
		var charIDCopy *int
		if rec.CharacterID != nil {
			val := *rec.CharacterID
			charIDCopy = &val
		}
		recCopy := DiagnosticRecord{
			Seq:         rec.Seq,
			Timestamp:   rec.Timestamp,
			Severity:    rec.Severity,
			Scope:       rec.Scope,
			Event:       rec.Event,
			Message:     rec.Message,
			CharacterID: charIDCopy,
			Revision:    rec.Revision,
		}
		matched = append(matched, recCopy)
	}

	hasMore := false
	var records []DiagnosticRecord
	if len(matched) > limit {
		records = matched[:limit]
		hasMore = true
	} else {
		records = matched
	}

	if records == nil {
		records = []DiagnosticRecord{}
	}

	var nextCursor string
	if len(records) > 0 {
		nextCursor = strconv.FormatUint(records[len(records)-1].Seq, 10)
	} else {
		if cursorExpired {
			nextCursor = strconv.FormatUint(newestSeq, 10)
		} else if hasCursor {
			nextCursor = cursor
		} else {
			nextCursor = strconv.FormatUint(newestSeq, 10)
		}
	}

	return DiagnosticLogResult{
		SaveSessionID:         saveSessionID,
		Records:               records,
		NextCursor:            nextCursor,
		HasMore:               hasMore,
		TotalBuffered:         totalBuffered,
		CursorExpired:         cursorExpired,
		OldestAvailableCursor: oldestCursorStr,
	}, nil
}
