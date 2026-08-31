import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  createTestQueryClient,
  makeDiagnosticsPort,
  stubCleanValidationReport,
  stubSaveCharacters,
  stubSaveSession,
  TestProviders,
} from "../../test/renderWithProviders";
import type { DiagnosticsPort, SaveValidationReportRequest } from "./diagnosticsPort";
import { useSaveValidationReports } from "./useSaveValidationReports";

const activeCharacters = stubSaveCharacters.characters.filter((character) => character.active);

function wrapperFor(diagnosticsPort: DiagnosticsPort) {
  return ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={createTestQueryClient()} diagnosticsPort={diagnosticsPort}>
      {children}
    </TestProviders>
  );
}

function render(port: DiagnosticsPort, saveRevision = stubSaveSession.saveRevision) {
  return renderHook(
    () => useSaveValidationReports(stubSaveSession.saveSessionID, saveRevision, activeCharacters),
    { wrapper: wrapperFor(port) },
  );
}

function renderForRevision(port: DiagnosticsPort, saveRevision: string) {
  return renderHook(
    ({ revision }: { revision: string }) =>
      useSaveValidationReports(stubSaveSession.saveSessionID, revision, activeCharacters),
    { wrapper: wrapperFor(port), initialProps: { revision: saveRevision } },
  );
}

describe("useSaveValidationReports", () => {
  it("reads a report that belongs to the session, the revision and the slot as clean", async () => {
    const { result } = render(makeDiagnosticsPort());

    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(result.current.reports).toEqual([{ ...stubCleanValidationReport, characterID: 0 }]);
  });

  // The adjacent boundary of the case below: a report whose numbers are exactly
  // as clean, and whose identity differs in one field, may not be accepted.
  it("never accepts a report that answers about another save state", async () => {
    for (const [label, report] of [
      ["revision", { ...stubCleanValidationReport, saveRevision: "1" }],
      ["session", { ...stubCleanValidationReport, saveSessionID: "session-2" }],
      ["slot", { ...stubCleanValidationReport, characterID: 4 }],
      ["active slot", { ...stubCleanValidationReport, active: false }],
    ] as const) {
      const { result } = render(
        makeDiagnosticsPort({ getSaveValidationReport: () => Promise.resolve(report) }),
      );

      // A stale answer is not a weaker verdict on this save: it is no verdict at
      // all, so it can never produce `clean` or `warnings`.
      await waitFor(() => expect(result.current.state, label).toBe("failed"));
      expect(result.current.reports, label).toEqual([]);
      expect(result.current.errorCount, label).toBe(0);
      expect(result.current.warningCount, label).toBe(0);
    }
  });

  it("asks again for a new revision and never reuses the answer about the old one", async () => {
    // The report of one revision is a complete answer about that revision and
    // about nothing else. A later revision is a different question, so it has to
    // reach the backend rather than be served from the previous answer.
    const getSaveValidationReport = vi.fn(
      ({ saveSessionID, characterID }: SaveValidationReportRequest) =>
        Promise.resolve({
          ...stubCleanValidationReport,
          saveSessionID,
          characterID,
          saveRevision,
        }),
    );
    let saveRevision = "0";
    const { result, rerender } = renderForRevision(
      makeDiagnosticsPort({ getSaveValidationReport }),
      "0",
    );

    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(result.current.reports).toEqual([
      { ...stubCleanValidationReport, characterID: 0, saveRevision: "0" },
    ]);
    expect(getSaveValidationReport).toHaveBeenCalledTimes(1);

    saveRevision = "1";
    rerender({ revision: "1" });

    // The cached revision 0 report may never stand in as the verdict on
    // revision 1, whatever its counters say.
    await waitFor(() => expect(getSaveValidationReport).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(result.current.saveRevision).toBe("1");
    expect(result.current.reports).toEqual([
      { ...stubCleanValidationReport, characterID: 0, saveRevision: "1" },
    ]);
  });

  it("never serves a cached report of the previous revision as the new verdict", async () => {
    // The backend keeps answering about revision 0 while the flow already
    // expects revision 1: the answer is complete, clean and about another save
    // state, so it is no verdict at all.
    const { result, rerender } = renderForRevision(
      makeDiagnosticsPort({
        getSaveValidationReport: () => Promise.resolve(stubCleanValidationReport),
      }),
      "0",
    );
    await waitFor(() => expect(result.current.state).toBe("clean"));

    rerender({ revision: "1" });

    await waitFor(() => expect(result.current.state).toBe("failed"));
    expect(result.current.reports).toEqual([]);
    expect(result.current.errorCount).toBe(0);
    expect(result.current.warningCount).toBe(0);
  });

  it("waits instead of judging while the session revision is unknown", () => {
    // A default parameter would swallow an explicitly passed `undefined`, so the
    // unknown-revision case is rendered directly.
    const { result } = renderHook(
      () => useSaveValidationReports(stubSaveSession.saveSessionID, undefined, activeCharacters),
      { wrapper: wrapperFor(makeDiagnosticsPort()) },
    );

    expect(result.current.state).toBe("pending");
    expect(result.current.reports).toEqual([]);
  });
});
