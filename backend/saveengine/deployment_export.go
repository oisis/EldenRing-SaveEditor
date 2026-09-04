package saveengine

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// DeploymentExportResult reports a save prepared for deployment.
type DeploymentExportResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	Platform      string `json:"platform"`
	Target        string `json:"target"`
}

// ExportForDeployment performs the shared safe preparation phase of section 3
// of the deployment specification: it serialises the current in-memory session
// state, validates the serialised bytes, writes them to a temporary file and
// verifies the written file by reloading it.
//
// It deliberately shares the whole path with Save: the same review
// authorisation, the same serialisation, the same validation and the same
// atomic write. A deployment can therefore never send a save that an ordinary
// Save would have refused.
//
// What it does not do is the point of the method. It does not touch
// session.sourcePath or sourceKind, does not clear the operation history, undo,
// redo or the review authorisation, does not mark the session clean and does
// not advance the revision. The session is exactly as dirty afterwards as it was
// before, because the local file the user owns was not written.
func (engine *Engine) ExportForDeployment(
	saveSessionID string,
	expectedRevision string,
	validationToken string,
	confirmWarnings bool,
	confirmBanRisk bool,
	target string,
) (DeploymentExportResult, error) {
	if engine == nil {
		return DeploymentExportResult{}, errors.New("saveengine.Engine is required")
	}
	if !isCanonicalRevision(expectedRevision) {
		return DeploymentExportResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if saveSessionID == "" {
		return DeploymentExportResult{}, apperror.MissingField("saveSessionID")
	}
	if target == "" {
		return DeploymentExportResult{}, apperror.MissingField("target")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return DeploymentExportResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	session := loaded.session
	if expectedRevision != session.revisionString() {
		return DeploymentExportResult{}, apperror.RevisionConflict(
			expectedRevision, session.revisionString())
	}
	if err := requireReviewAuthorization(
		session, expectedRevision, validationToken, confirmWarnings, confirmBanRisk); err != nil {
		return DeploymentExportResult{}, err
	}

	candidate, err := serializeContainer(loaded)
	if err != nil {
		return DeploymentExportResult{}, fmt.Errorf("cannot serialize save session: %w", err)
	}
	if err := validateSerialized(candidate, session.platform); err != nil {
		return DeploymentExportResult{}, fmt.Errorf("cannot validate serialized save: %w", err)
	}
	if err := writeAtomically(target, candidate); err != nil {
		return DeploymentExportResult{}, fmt.Errorf("cannot write the prepared save: %w", err)
	}
	if err := verifyWrittenTarget(target, candidate, session.platform); err != nil {
		return DeploymentExportResult{}, fmt.Errorf("the prepared save failed final validation: %w", err)
	}
	return DeploymentExportResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  expectedRevision,
		Platform:      string(session.platform),
		Target:        target,
	}, nil
}
