import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { queryKeys } from "../queryKeys";
import { useSettingsPort } from "./settingsClient";
import type {
  DiagnosticEvent,
  DiagnosticMode,
  DiagnosticReportResult,
  HostLocation,
  HostSettings,
} from "./settingsPort";

/**
 * The persistent host settings as the interface reads them.
 *
 * They are backend state and therefore live outside every session key. The
 * query is not retried: a failing desktop bridge call does not become healthy
 * by repeating it.
 */
export function useHostSettings() {
  const port = useSettingsPort();

  return useQuery({
    queryKey: queryKeys.hostSettings(),
    queryFn: () => port.getHostSettings(),
    retry: false,
  });
}

/**
 * Stores the complete host settings value.
 *
 * The answer of the call that changed the setting is what updates the cache, so
 * the interface never renders a setting the backend did not confirm. This is a
 * host-settings mutation and deliberately not a save mutation: no revision
 * moves and no mutation receipt exists, so nothing here touches a session key.
 */
export function useSetHostSettings() {
  const port = useSettingsPort();
  const queryClient = useQueryClient();

  return useMutation<
    HostSettings,
    Error,
    { skipReviewForNormalRisk: boolean; remoteBackupPolicy: string }
  >({
    mutationFn: (settings) => port.setHostSettings(settings),
    onSuccess: (settings) => queryClient.setQueryData(queryKeys.hostSettings(), settings),
  });
}

/** Opens one known host directory, as an explicit user action. */
export function useOpenHostLocation() {
  const port = useSettingsPort();

  return useMutation<void, Error, HostLocation>({
    mutationFn: (location) => port.openHostLocation(location),
  });
}

/** Exports the redacted diagnostic report through the native Save As dialog. */
export function useExportDiagnosticReport() {
  const port = useSettingsPort();

  return useMutation<DiagnosticReportResult, Error, string | undefined>({
    mutationFn: (saveSessionID) => port.exportDiagnosticReport(saveSessionID),
  });
}

/**
 * The runtime diagnostic state.
 *
 * It is instance state rather than persistent settings, so it lives outside
 * every session key and is never seeded from a stored value: a fresh launch
 * always reports Debug Mode as disabled until the backend says otherwise.
 */
export function useDiagnosticMode() {
  const port = useSettingsPort();

  return useQuery({
    queryKey: queryKeys.diagnosticMode(),
    queryFn: () => port.getDiagnosticMode(),
    retry: false,
    refetchInterval: 1000,
  });
}

/**
 * Turns extended diagnostics on or off.
 *
 * The answer of the call that changed the flag is what updates the cache, so
 * the checkbox never shows a state the backend did not confirm. It is not a
 * save mutation: no revision moves, no receipt exists and no session key is
 * touched.
 */
export function useSetDiagnosticMode() {
  const port = useSettingsPort();
  const queryClient = useQueryClient();

  return useMutation<DiagnosticMode, Error, boolean>({
    mutationFn: (enabled) => port.setDiagnosticMode(enabled),
    onSuccess: (mode) => queryClient.setQueryData(queryKeys.diagnosticMode(), mode),
  });
}

/** How many records the console keeps rendered at once. */
const consoleRecordLimit = 200;

/** How often an expanded console asks for what it has not seen yet. */
const consolePollIntervalMS = 1000;

/**
 * The instance-wide diagnostic stream as the bottom console reads it.
 *
 * It polls only while `active`, which is exactly while the console is
 * expanded and mounted, and it asks for records after the cursor it already
 * holds, so a record is never rendered twice. An expired cursor means the
 * records in between were evicted: the console restarts from the page the
 * backend offered instead of showing a silent gap.
 */
export function useDiagnosticEvents(active: boolean) {
  const port = useSettingsPort();
  const [records, setRecords] = useState<readonly DiagnosticEvent[]>([]);
  const [failed, setFailed] = useState(false);
  const cursor = useRef("");
  const generation = useRef(0);
  useEffect(() => {
    const current = ++generation.current;
    if (!active) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const poll = async () => {
      try {
        const page = await port.getDiagnosticEvents({
          cursor: cursor.current,
          limit: consoleRecordLimit,
          severity: "",
        });
        if (generation.current !== current) return;
        cursor.current = page.nextCursor;
        setFailed(false);
        if (page.records.length === 0 && !page.cursorExpired) {
          return;
        }
        setRecords((previous) => {
          const kept = page.cursorExpired ? [] : previous;
          const unique = new Map([...kept, ...page.records].map((record) => [record.seq, record]));
          return [...unique.values()].slice(-consoleRecordLimit);
        });
      } catch {
        // The console reports that it cannot read the stream. It never renders
        // the transport's own message: that text carries bridge internals.
        if (generation.current === current) setFailed(true);
      } finally {
        if (generation.current === current)
          timer = setTimeout(() => void poll(), consolePollIntervalMS);
      }
    };
    void poll();
    return () => {
      generation.current++;
      clearTimeout(timer);
    };
  }, [active, port]);

  return { records, failed };
}
