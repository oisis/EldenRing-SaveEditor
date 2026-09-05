package desktop

import (
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	diagnosticsservice "github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	deploymentendpoints "github.com/oisis/EldenRing-SaveForge/backend/endpoints/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// This file carries the Tools half of the bridge surface: the host settings,
// the Build Templates library, deployment, Save Manager and About & Updates.
//
// Every method here delegates to a public backend endpoint exactly like the
// rest of the bridge. The two things it owns beyond that are host capabilities
// no endpoint can have: opening a location in the host shell, and cancelling a
// running deployment operation.

// deploymentProgressEvent is the host event a running deployment reports on.
const deploymentProgressEvent = "deployment.progress"

// publishDeploymentProgress forwards one progress report to the frontend.
// Emission before the host has started is dropped rather than buffered: the
// operation's own result still states every stage it performed.
func (b *Bridge) publishDeploymentProgress(progress deployment.Progress) {
	b.observeStage(progress)
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return
	}
	emit := b.emitEvent
	if emit == nil {
		emit = runtime.EventsEmit
	}
	emit(ctx, deploymentProgressEvent, progress)
}

// operationContext registers a cancellable context for one long operation and
// returns it together with its release function.
func (b *Bridge) operationContext(operationID string) (context.Context, func()) {
	parent := b.hostContextOrNil()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	if operationID == "" {
		return ctx, cancel
	}
	b.operationsMutex.Lock()
	if b.operations == nil {
		b.operations = map[string]context.CancelFunc{}
	}
	if b.operationCorrelations == nil {
		b.operationCorrelations = map[string]string{}
	}
	b.operations[operationID] = cancel
	b.operationCorrelations[operationID] = diagnosticsservice.NewCorrelationID()
	b.operationsMutex.Unlock()
	return ctx, func() {
		b.operationsMutex.Lock()
		delete(b.operations, operationID)
		delete(b.operationCorrelations, operationID)
		b.operationsMutex.Unlock()
		cancel()
	}
}

// correlationFor resolves the generated correlation value of one running
// operation. An unknown identifier yields an empty value rather than being
// echoed into a record: the caller's own string is never logged.
func (b *Bridge) correlationFor(operationID string) string {
	if operationID == "" {
		return ""
	}
	b.operationsMutex.Lock()
	defer b.operationsMutex.Unlock()
	return b.operationCorrelations[operationID]
}

// CancelDeploymentOperation cooperatively cancels one running operation.
// Cancelling an operation that already finished, or one that never existed, is
// not an error: the caller learns the real outcome from the operation's result.
func (b *Bridge) CancelDeploymentOperation(operationID string) error {
	b.operationsMutex.Lock()
	cancel, running := b.operations[operationID]
	b.operationsMutex.Unlock()
	if running {
		cancel()
	}
	return nil
}

// GetHostSettings delegates to the GetHostSettings endpoint.
func (b *Bridge) GetHostSettings() (application.HostSettingsResult, error) {
	return bridged(application.GetHostSettings(b.hostSettings, b.diagnostics))
}

// SetHostSettings delegates to the SetHostSettings endpoint.
func (b *Bridge) SetHostSettings(
	skipReviewForNormalRisk bool, remoteBackupPolicy string,
) (application.HostSettingsResult, error) {
	return bridged(application.SetHostSettings(
		b.hostSettings, b.diagnostics, skipReviewForNormalRisk, remoteBackupPolicy))
}

// Host locations are identifiers, not paths. Each one resolves to a directory
// its own backend owner supplies: the settings store owns the configuration
// directory and the diagnostic service owns the log directory.
const (
	hostLocationConfiguration = "configuration"
	hostLocationLogs          = "logs"
)

// OpenHostLocation opens a known host directory in the file manager as an
// explicit user action.
//
// It accepts an identifier and never a path. There is deliberately no bridge
// method that opens an arbitrary location the frontend supplies, so no frontend
// defect and no injected string can make the host open something else.
func (b *Bridge) OpenHostLocation(location string) error {
	if b.hostSettings == nil {
		return bridgeError(errors.New("the host settings store is not available"))
	}
	path := ""
	switch location {
	case hostLocationConfiguration:
		path = b.hostSettings.Directory()
	case hostLocationLogs:
		path = b.diagnostics.Directory()
	default:
		return bridgeError(errors.New("unknown host location"))
	}
	if path == "" {
		return bridgeError(errors.New("this host has no such directory"))
	}
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return bridgeError(errors.New("the desktop host is not started yet"))
	}
	b.openURL(ctx, "file://"+path)
	return nil
}

// OpenProjectLink opens one approved project address in the host's default
// browser, as an explicit user action. The identifier is resolved by the
// backend allowlist; an unknown one is refused.
func (b *Bridge) OpenProjectLink(linkID string) error {
	url, err := application.ResolveProjectLink(linkID)
	if err != nil {
		return bridgeError(err)
	}
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return bridgeError(errors.New("the desktop host is not started yet"))
	}
	b.openURL(ctx, url)
	return nil
}

// GetProjectLinks delegates to the GetProjectLinks endpoint.
func (b *Bridge) GetProjectLinks() (application.GetProjectLinksResult, error) {
	return bridged(application.GetProjectLinks())
}

// CheckForUpdates delegates to the CheckForUpdates endpoint. The endpoint's test
// override is never forwarded: the frontend cannot redirect the check.
func (b *Bridge) CheckForUpdates() (application.CheckForUpdatesResult, error) {
	ctx := b.hostContextOrNil()
	if ctx == nil {
		ctx = context.Background()
	}
	return bridged(application.CheckForUpdates(ctx, b.applicationVersion, ""))
}

// ExportDiagnosticReport opens the native Save As dialog and writes the redacted
// report to the path the user chose. A cancelled dialog is an ordinary outcome:
// nothing is written and no error is reported.
func (b *Bridge) ExportDiagnosticReport(saveSessionID string) (application.DiagnosticReportResult, error) {
	target, err := b.SelectSaveTarget("saveforge-diagnostics.json")
	if err != nil {
		return application.DiagnosticReportResult{}, err
	}
	if target == "" {
		return application.DiagnosticReportResult{Exported: false}, nil
	}
	return bridged(application.ExportDiagnosticReport(
		b.applicationVersion, b.hostSettings, b.diagnostics, b.saveEngine,
		saveSessionID, target))
}

func (b *Bridge) openURL(ctx context.Context, url string) {
	open := b.openHostURL
	if open == nil {
		open = runtime.BrowserOpenURL
	}
	open(ctx, url)
}

// GetBuildTemplates delegates to the GetBuildTemplates endpoint.
func (b *Bridge) GetBuildTemplates(
	search string, tags []string, page int, pageSize int,
) (templates.GetBuildTemplatesResult, error) {
	return bridged(templates.GetBuildTemplates(b.buildTemplates, search, tags, page, pageSize))
}

// GetBuildTemplate delegates to the GetBuildTemplate endpoint.
func (b *Bridge) GetBuildTemplate(templateID string) (templates.GetBuildTemplateResult, error) {
	return bridged(templates.GetBuildTemplate(b.buildTemplates, templateID))
}

// GetBuildTemplatePreview delegates to the GetBuildTemplatePreview endpoint.
func (b *Bridge) GetBuildTemplatePreview(
	request templates.GetBuildTemplatePreviewRequest,
) (templates.GetBuildTemplatePreviewResult, error) {
	return bridged(templates.GetBuildTemplatePreview(
		b.buildTemplates, b.saveEngine, b.gameCatalog, request))
}

// ApplyBuildTemplate delegates to the ApplyBuildTemplate endpoint. Its result
// embeds the shared mutation receipt, so applying a template refreshes the
// session through exactly the same path as every other save mutation.
func (b *Bridge) ApplyBuildTemplate(
	request templates.ApplyBuildTemplateRequest,
) (templates.ApplyBuildTemplateResult, error) {
	return bridged(templates.ApplyBuildTemplate(
		b.buildTemplates, b.saveEngine, b.gameCatalog, request))
}

// CreateBuildTemplate captures every field the current schema can safely
// export. The desktop screen offers one "from active character" action rather
// than a section picker, so the bridge builds the explicit selection required
// by the backend instead of relying on the generated representation of
// SectionSelection (whose custom JSON shape has no generated fields).
func (b *Bridge) CreateBuildTemplate(
	saveSessionID string,
	sourceCharacterID int,
	name string,
	description string,
	tags []string,
) (templates.CreateBuildTemplateResult, error) {
	return bridged(templates.CreateBuildTemplate(
		b.buildTemplates, b.saveEngine, b.gameCatalog, b.applicationVersion,
		templates.CreateBuildTemplateRequest{
			SaveSessionID:     saveSessionID,
			SourceCharacterID: sourceCharacterID,
			Selection:         desktopBuildTemplateSelection(),
			Name:              name,
			Description:       description,
			Tags:              tags,
		}))
}

func desktopBuildTemplateSelection() buildtemplates.TemplateSelection {
	return buildtemplates.TemplateSelection{
		Profile: &buildtemplates.SectionSelection{Fields: map[string]bool{
			"name": true, "level": true,
		}},
		Stats: &buildtemplates.SectionSelection{Fields: map[string]bool{
			"vigor": true, "mind": true, "endurance": true, "strength": true,
			"dexterity": true, "intelligence": true, "faith": true, "arcane": true,
		}},
		Spells: &buildtemplates.SectionSelection{Fields: map[string]bool{
			"spell1": true, "spell2": true, "spell3": true, "spell4": true,
			"spell5": true, "spell6": true, "spell7": true, "spell8": true,
			"spell9": true, "spell10": true, "spell11": true, "spell12": true,
		}},
	}
}

// UpdateBuildTemplate delegates to the UpdateBuildTemplate endpoint.
func (b *Bridge) UpdateBuildTemplate(
	templateID string, request templates.UpdateBuildTemplateRequest,
) (templates.UpdateBuildTemplateResult, error) {
	return bridged(templates.UpdateBuildTemplate(b.buildTemplates, templateID, request))
}

// DeleteBuildTemplate delegates to the DeleteBuildTemplate endpoint.
func (b *Bridge) DeleteBuildTemplate(
	templateID string, templateRevision string,
) (templates.DeleteBuildTemplateResult, error) {
	return bridged(templates.DeleteBuildTemplate(b.buildTemplates, templateID, templateRevision))
}

// ImportBuildTemplate opens the native document dialog and stores the chosen
// Build Template. A cancelled dialog is an ordinary outcome and stores nothing.
func (b *Bridge) ImportBuildTemplate() (templates.ImportBuildTemplateResult, error) {
	if b.chooseDocument == nil {
		return templates.ImportBuildTemplateResult{},
			bridgeError(errors.New("the native document dialog is not available"))
	}
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return templates.ImportBuildTemplateResult{},
			bridgeError(errors.New("the desktop host is not started yet"))
	}
	source, err := b.chooseDocument(ctx)
	if err != nil {
		return templates.ImportBuildTemplateResult{}, bridgeError(err)
	}
	if source == "" {
		return templates.ImportBuildTemplateResult{}, nil
	}
	return bridged(templates.ImportBuildTemplate(b.buildTemplates, source))
}

// GetDeploymentTargets delegates to the GetDeploymentTargets endpoint.
func (b *Bridge) GetDeploymentTargets() (deploymentendpoints.GetDeploymentTargetsResult, error) {
	return bridged(deploymentendpoints.GetDeploymentTargets(b.deploymentStore))
}

// CreateDeploymentTarget delegates to the CreateDeploymentTarget endpoint.
func (b *Bridge) CreateDeploymentTarget(
	input deploymentendpoints.TargetInput,
) (deploymentendpoints.GetDeploymentTargetsResult, error) {
	return bridged(deploymentendpoints.CreateDeploymentTarget(b.deploymentStore, input))
}

// UpdateDeploymentTarget delegates to the UpdateDeploymentTarget endpoint.
func (b *Bridge) UpdateDeploymentTarget(
	input deploymentendpoints.TargetInput,
) (deploymentendpoints.GetDeploymentTargetsResult, error) {
	return bridged(deploymentendpoints.UpdateDeploymentTarget(b.deploymentStore, input))
}

// DeleteDeploymentTarget delegates to the DeleteDeploymentTarget endpoint.
func (b *Bridge) DeleteDeploymentTarget(
	targetID string,
) (deploymentendpoints.GetDeploymentTargetsResult, error) {
	return bridged(deploymentendpoints.DeleteDeploymentTarget(b.deploymentStore, targetID))
}

// TestDeploymentTarget delegates to the TestDeploymentTarget endpoint.
func (b *Bridge) TestDeploymentTarget(
	targetID string,
) (deploymentendpoints.TestDeploymentTargetResult, error) {
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.TestDeploymentTarget(ctx, b.deploymentService, targetID))
}

// TrustDeploymentHostKey delegates to the TrustDeploymentHostKey endpoint.
//
// The fingerprint the frontend passes is the one TestDeploymentTarget reported
// as observed. The backend accepts it only if a handshake with that target's
// address actually presented it, so approving an invented value is impossible
// however the call is made.
func (b *Bridge) TrustDeploymentHostKey(
	targetID string, fingerprint string,
) (deploymentendpoints.GetDeploymentTargetsResult, error) {
	return bridged(deploymentendpoints.TrustDeploymentHostKey(
		b.deploymentStore, targetID, fingerprint))
}

// ForgetDeploymentHostKey delegates to the ForgetDeploymentHostKey endpoint.
func (b *Bridge) ForgetDeploymentHostKey(
	targetID string,
) (deploymentendpoints.GetDeploymentTargetsResult, error) {
	return bridged(deploymentendpoints.ForgetDeploymentHostKey(b.deploymentStore, targetID))
}

// GetDeploymentGameStatus delegates to the GetDeploymentGameStatus endpoint.
func (b *Bridge) GetDeploymentGameStatus(
	targetID string,
) (deploymentendpoints.GetDeploymentGameStatusResult, error) {
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.GetDeploymentGameStatus(ctx, b.deploymentService, targetID))
}

// LaunchTargetGame delegates to the LaunchTargetGame endpoint.
func (b *Bridge) LaunchTargetGame(targetID string) (deploymentendpoints.CommandOutcome, error) {
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.LaunchTargetGame(ctx, b.deploymentService, targetID))
}

// CloseTargetGame delegates to the CloseTargetGame endpoint.
func (b *Bridge) CloseTargetGame(targetID string) (deploymentendpoints.CommandOutcome, error) {
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.CloseTargetGame(ctx, b.deploymentService, targetID))
}

// DeployToTarget delegates to the DeployToTarget endpoint. The operation is
// registered under its identifier so CancelDeploymentOperation can reach it.
func (b *Bridge) DeployToTarget(
	request deploymentendpoints.DeployRequest,
) (deploymentendpoints.OperationResult, error) {
	ctx, release := b.operationContext(request.OperationID)
	defer release()
	logger := b.beginOperation(
		diagnosticsservice.OperationDeployToTarget, b.correlationFor(request.OperationID))
	result, err := deploymentendpoints.DeployToTarget(
		ctx, b.deploymentService, b.saveEngine, request)
	logger.finishOperationResult(result, err)
	return bridged(result, err)
}

// DownloadFromTarget delegates to the DownloadFromTarget endpoint.
func (b *Bridge) DownloadFromTarget(
	request deploymentendpoints.DownloadRequest,
) (deploymentendpoints.OperationResult, error) {
	ctx, release := b.operationContext(request.OperationID)
	defer release()
	logger := b.beginOperation(
		diagnosticsservice.OperationDownloadFromTarget, b.correlationFor(request.OperationID))
	result, err := deploymentendpoints.DownloadFromTarget(ctx, b.deploymentService, request)
	logger.finishOperationResult(result, err)
	if err == nil && result.Completed && result.LocalPath != "" {
		b.trackStagedDownload(result.LocalPath)
	}
	return bridged(result, err)
}

// GetTargetBackups delegates to the GetTargetBackups endpoint.
func (b *Bridge) GetTargetBackups(
	targetID string,
) (deploymentendpoints.GetTargetBackupsResult, error) {
	return bridged(deploymentendpoints.GetTargetBackups(b.deploymentStore, targetID))
}

// CreateTargetBackup delegates to the CreateTargetBackup endpoint.
func (b *Bridge) CreateTargetBackup(
	targetID string, tags []string, description string,
) (deploymentendpoints.GetTargetBackupsResult, error) {
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.CreateTargetBackup(
		ctx, b.deploymentService, b.deploymentStore, targetID, tags, description))
}

// ActivateTargetBackup delegates to the ActivateTargetBackup endpoint.
func (b *Bridge) ActivateTargetBackup(
	operationID string,
	targetID string,
	backupID string,
	continueWithUnknownGameStatus bool,
	confirmRemoteBackup bool,
) (deploymentendpoints.ActivateTargetBackupResult, error) {
	ctx, release := b.operationContext(operationID)
	defer release()
	logger := b.beginOperation(
		diagnosticsservice.OperationActivateBackup, b.correlationFor(operationID))
	result, err := deploymentendpoints.ActivateTargetBackup(
		ctx, b.deploymentService, b.deploymentStore, operationID, targetID, backupID,
		continueWithUnknownGameStatus, confirmRemoteBackup)
	logger.finishOperationResult(result.Operation, err)
	return bridged(result, err)
}

// ClearActiveTargetBackup delegates to the ClearActiveTargetBackup endpoint.
func (b *Bridge) ClearActiveTargetBackup(
	targetID string,
) (deploymentendpoints.GetTargetBackupsResult, error) {
	return bridged(deploymentendpoints.ClearActiveTargetBackup(
		b.deploymentService, b.deploymentStore, targetID))
}

// UpdateTargetBackup delegates to the UpdateTargetBackup endpoint.
func (b *Bridge) UpdateTargetBackup(
	targetID string, backupID string, tags []string, description string,
) (deploymentendpoints.GetTargetBackupsResult, error) {
	return bridged(deploymentendpoints.UpdateTargetBackup(
		b.deploymentStore, targetID, backupID, tags, description))
}

// DeleteTargetBackup delegates to the DeleteTargetBackup endpoint.
func (b *Bridge) DeleteTargetBackup(
	targetID string, backupID string,
) (deploymentendpoints.GetTargetBackupsResult, error) {
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.DeleteTargetBackup(
		ctx, b.deploymentService, b.deploymentStore, targetID, backupID))
}

// DownloadTargetBackup opens the native Save As dialog and copies the chosen
// backup to the path the user agreed to. A cancelled dialog writes nothing and
// is not an error, so a download can never silently overwrite a file.
func (b *Bridge) DownloadTargetBackup(
	targetID string, backupID string, suggestedName string,
) (deploymentendpoints.DownloadTargetBackupResult, error) {
	target, err := b.SelectSaveTarget(suggestedName)
	if err != nil {
		return deploymentendpoints.DownloadTargetBackupResult{}, err
	}
	if target == "" {
		return deploymentendpoints.DownloadTargetBackupResult{}, nil
	}
	ctx, release := b.operationContext("")
	defer release()
	return bridged(deploymentendpoints.DownloadTargetBackup(
		ctx, b.deploymentService, targetID, backupID, target))
}
