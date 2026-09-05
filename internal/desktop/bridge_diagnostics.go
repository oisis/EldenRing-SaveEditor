package desktop

import (
	"context"
	"errors"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
	deploymentdomain "github.com/oisis/EldenRing-SaveForge/backend/deployment"
	diagnosticsservice "github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
)

// This file carries the diagnostics half of the bridge surface and the only
// place operation events are emitted.
//
// The bridge is where every shared operation boundary already meets: opening a
// save, Save and Save As, ApplyRepairs, deploy, download and backup activation
// all pass through exactly one method here. Emitting from this one layer keeps
// SaveEngine, the writers and the deployment service free of logging, which is
// also what keeps file I/O out from under the engine lock.

// SetDiagnosticMode delegates to the SetDiagnosticMode endpoint.
func (b *Bridge) SetDiagnosticMode(enabled bool) (diagnostics.DiagnosticModeResult, error) {
	return bridged(diagnostics.SetDiagnosticMode(b.diagnostics, enabled))
}

// GetDiagnosticMode delegates to the GetDiagnosticMode endpoint.
func (b *Bridge) GetDiagnosticMode() (diagnostics.DiagnosticModeResult, error) {
	return bridged(diagnostics.GetDiagnosticMode(b.diagnostics))
}

// GetDiagnosticEvents delegates to the GetDiagnosticEvents endpoint. It is the
// instance-wide stream the bottom console reads and needs no open save.
func (b *Bridge) GetDiagnosticEvents(
	cursor string, limit int, severity string,
) (diagnostics.GetDiagnosticEventsResult, error) {
	return bridged(diagnostics.GetDiagnosticEvents(b.diagnostics, cursor, limit, severity))
}

// operationLogger closes over one execution of one shared operation boundary.
type operationLogger struct {
	bridge        *Bridge
	operation     string
	correlationID string
	started       time.Time
}

type diagnosticTiming struct {
	stage     string
	started   time.Time
	durations map[string]int64
}

// observeStage measures progress intervals only. Completion is confirmed later
// by the operation result, not inferred from a progress announcement.
func (b *Bridge) observeStage(progress deploymentdomain.Progress) {
	b.operationsMutex.Lock()
	defer b.operationsMutex.Unlock()
	correlation := b.operationCorrelations[progress.OperationID]
	timing := b.operationTimings[correlation]
	if timing == nil {
		return
	}
	if !progress.Finished && progress.Stage == timing.stage {
		return
	}
	now := time.Now()
	if timing.stage != "" {
		timing.durations[timing.stage] += now.Sub(timing.started).Milliseconds()
	}
	timing.stage = progress.Stage
	timing.started = now
	if progress.Finished {
		timing.stage = ""
	}
}

// beginOperation records operation_started and returns the logger that will
// record its outcome. The started record is debug, so an ordinary run adds
// nothing to the log until Debug Mode is on.
func (b *Bridge) beginOperation(operation string, correlationID string) operationLogger {
	if correlationID == "" {
		correlationID = diagnosticsservice.NewCorrelationID()
	}
	logger := operationLogger{
		bridge:        b,
		operation:     operation,
		correlationID: correlationID,
		started:       time.Now(),
	}
	b.operationsMutex.Lock()
	if b.operationTimings == nil {
		b.operationTimings = map[string]*diagnosticTiming{}
	}
	b.operationTimings[correlationID] = &diagnosticTiming{durations: map[string]int64{}}
	b.operationsMutex.Unlock()
	b.diagnostics.Log(diagnosticsservice.Entry{
		Event:         diagnosticsservice.EventOperationStarted,
		Operation:     operation,
		CorrelationID: correlationID,
	})
	return logger
}

// finish records operation_finished with the outcome the operation actually
// reported. Only closed identifiers reach the record: a status, an error code
// and a real target state, never an error's own text.
func (logger operationLogger) finish(status string, code string, targetState string) {
	logger.bridge.operationsMutex.Lock()
	delete(logger.bridge.operationTimings, logger.correlationID)
	logger.bridge.operationsMutex.Unlock()
	logger.bridge.diagnostics.Log(diagnosticsservice.Entry{
		Event:         diagnosticsservice.EventOperationFinished,
		Operation:     logger.operation,
		CorrelationID: logger.correlationID,
		Status:        status,
		Code:          code,
		TargetState:   targetState,
		DurationMS:    time.Since(logger.started).Milliseconds(),
	})
}

// finishWithError is the outcome of a plain save-side operation: it succeeded,
// or it failed with one classified application error code.
func (logger operationLogger) finishWithError(err error) {
	if err == nil {
		logger.finish(diagnosticsservice.StatusSucceeded, "", "")
		return
	}
	if errors.Is(err, context.Canceled) {
		logger.finish(diagnosticsservice.StatusCancelled, deploymentdomain.BlockedCancelled, "")
		return
	}
	if apperror.From(err).Code == apperror.CodeRevisionConflict {
		logger.finish(diagnosticsservice.StatusBlocked, apperror.CodeRevisionConflict, "")
		return
	}
	logger.finish(diagnosticsservice.StatusFailed, apperror.From(err).Code, "")
}

// finishOperationResult is the outcome of a deployment operation, which
// reports a block, a failure and a real target state instead of an error.
func (logger operationLogger) finishOperationResult(
	result deploymentdomain.OperationResult, err error,
) {
	// Progress announces an attempted stage, not its success. Only the returned
	// outcome confirms completion. Do not log raw Detail strings.
	for _, stage := range result.Stages {
		if stage.Completed {
			duration := int64(0)
			logger.bridge.operationsMutex.Lock()
			if timing := logger.bridge.operationTimings[logger.correlationID]; timing != nil {
				duration = timing.durations[stage.Stage]
			}
			logger.bridge.operationsMutex.Unlock()
			logger.bridge.diagnostics.Log(diagnosticsservice.Entry{
				Event:     diagnosticsservice.EventOperationStageFinished,
				Operation: logger.operation, Stage: stage.Stage,
				CorrelationID: logger.correlationID,
				DurationMS:    duration,
			})
		}
	}
	if err != nil {
		logger.finish(diagnosticsservice.StatusFailed, apperror.From(err).Code, result.TargetState)
		return
	}
	switch {
	case result.Blocked == deploymentdomain.BlockedCancelled:
		logger.finish(diagnosticsservice.StatusCancelled, result.Blocked, result.TargetState)
	case result.Blocked != "":
		logger.finish(diagnosticsservice.StatusBlocked, result.Blocked, result.TargetState)
	case result.Failure != "":
		logger.finish(diagnosticsservice.StatusFailed, result.Failure, result.TargetState)
	default:
		logger.finish(diagnosticsservice.StatusSucceeded, "", result.TargetState)
	}
}

// loggedSaveOperation wraps one save-side boundary. It changes neither the
// result nor the error: a logger failure can never alter what the user's edit,
// save or repair actually did.
func loggedSaveOperation[T any](
	bridge *Bridge, operation string, call func() (T, error),
) (T, error) {
	logger := bridge.beginOperation(operation, "")
	value, err := call()
	logger.finishWithError(err)
	return bridged(value, err)
}
