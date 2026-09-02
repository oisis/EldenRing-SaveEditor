package saveengine

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

type reviewAuthorization struct {
	token           string
	revision        string
	warningRequired bool
	banRiskRequired bool
}

// ReviewValidationStage is one real completed validation stage. Percent is
// present only because this validator knows the exact number of stages.
type ReviewValidationStage struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
}

type ReviewValidationIssue struct {
	Code        string        `json:"code"`
	Severity    OperationRisk `json:"severity"`
	Message     string        `json:"message"`
	OperationID string        `json:"operationID,omitempty"`
}

// ReviewValidationResult authorizes saving only the exact revision it names.
// Any later mutation clears the authorization in the central commit path.
type ReviewValidationResult struct {
	SaveSessionID   string                  `json:"saveSessionID"`
	SaveRevision    string                  `json:"saveRevision"`
	ValidationToken string                  `json:"validationToken,omitempty"`
	Valid           bool                    `json:"valid"`
	WarningCount    int                     `json:"warningCount"`
	BanRiskCount    int                     `json:"banRiskCount"`
	CriticalCount   int                     `json:"criticalCount"`
	Stages          []ReviewValidationStage `json:"stages"`
	Issues          []ReviewValidationIssue `json:"issues"`
}

func newValidationToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cannot create validation token: %w", err)
	}
	return "validation-" + hex.EncodeToString(raw), nil
}

// ValidateReviewChanges validates an immutable copy of the current snapshot and
// binds the result to its exact revision. The caller may run this in a managed
// background operation; this method itself performs no frontend or Wails work.
func (engine *Engine) ValidateReviewChanges(
	saveSessionID string,
	expectedRevision string,
) (ReviewValidationResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return ReviewValidationResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if saveSessionID == "" {
		return ReviewValidationResult{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		engine.mutex.Unlock()
		return ReviewValidationResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if expectedRevision != loaded.session.revisionString() {
		current := loaded.session.revisionString()
		engine.mutex.Unlock()
		return ReviewValidationResult{}, apperror.RevisionConflict(expectedRevision, current)
	}
	platform := loaded.session.platform
	snapshot := append([]byte(nil), loaded.snapshot.data...)
	history := make([]operationEntry, len(loaded.operations))
	for index, entry := range loaded.operations {
		history[index] = cloneOperationEntry(entry)
	}
	engine.mutex.Unlock()

	result := ReviewValidationResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  expectedRevision,
		Stages: []ReviewValidationStage{
			{Stage: "history", Percent: 25},
			{Stage: "serialization", Percent: 50},
			{Stage: "reload", Percent: 75},
			{Stage: "validation", Percent: 100},
		},
		Issues: []ReviewValidationIssue{},
	}
	for _, entry := range history {
		switch entry.Record.Risk {
		case OperationRiskWarning:
			result.WarningCount++
		case OperationRiskBanRisk:
			result.BanRiskCount++
		case OperationRiskCritical:
			result.CriticalCount++
			result.Issues = append(result.Issues, ReviewValidationIssue{
				Code:        "critical_operation",
				Severity:    OperationRiskCritical,
				Message:     "A critical operation blocks saving.",
				OperationID: entry.Record.OperationID,
			})
		}
	}

	temporary := &loadedSave{
		session:  &Session{platform: platform},
		snapshot: &codec{data: snapshot},
	}
	candidate, err := serializeContainer(temporary)
	if err == nil {
		err = validateSerialized(candidate, platform)
	}
	if err != nil {
		result.CriticalCount++
		result.Issues = append(result.Issues, ReviewValidationIssue{
			Code:     "save_validation_failed",
			Severity: OperationRiskCritical,
			Message:  "The current save state failed reload validation.",
		})
		return result, nil
	}
	if result.CriticalCount > 0 {
		return result, nil
	}

	token, err := newValidationToken()
	if err != nil {
		return ReviewValidationResult{}, err
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists = engine.sessions[saveSessionID]
	if !exists {
		return ReviewValidationResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if expectedRevision != loaded.session.revisionString() {
		return ReviewValidationResult{}, apperror.RevisionConflict(
			expectedRevision, loaded.session.revisionString())
	}
	if fingerprintBytes(loaded.snapshot.data) != fingerprintBytes(snapshot) {
		return ReviewValidationResult{}, errors.New("save snapshot changed during validation")
	}
	loaded.session.reviewAuthorization = &reviewAuthorization{
		token:           token,
		revision:        expectedRevision,
		warningRequired: result.WarningCount > 0,
		banRiskRequired: result.BanRiskCount > 0,
	}
	result.Valid = true
	result.ValidationToken = token
	return result, nil
}

func requireReviewAuthorization(
	session *Session,
	expectedRevision string,
	token string,
	confirmWarnings bool,
	confirmBanRisk bool,
) error {
	if token == "" {
		return apperror.MissingField("validationToken")
	}
	authorization := session.reviewAuthorization
	if authorization == nil || authorization.token != token || authorization.revision != expectedRevision {
		return errors.New("Review Changes validation is missing or stale")
	}
	if authorization.warningRequired && !confirmWarnings {
		return errors.New("Review Changes warnings require explicit confirmation")
	}
	if authorization.banRiskRequired && !confirmBanRisk {
		return errors.New("Review Changes ban risk requires separate explicit confirmation")
	}
	return nil
}
