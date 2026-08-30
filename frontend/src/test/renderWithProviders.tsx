import { I18nProvider } from "@lingui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderResult, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { ApplicationInfoPortProvider } from "../application/application-info/applicationInfoClient";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../application/application-info/applicationInfoPort";
import { SaveSessionPortProvider } from "../application/save-session/saveSessionClient";
import type { SaveSession, SaveSessionPort } from "../application/save-session/saveSessionPort";
import { activateLocale, i18n, type Locale } from "../i18n/i18n";

/**
 * Components and hooks are exercised through the application ports, never
 * through a mock of the generated desktop bindings: such a test must fail when
 * the application contract breaks, not when the transport detail changes.
 */
export const stubApplicationInfo: ApplicationInfo = {
  version: "2.0.0-test",
  schemas: [{ name: "game_catalog", minimumVersion: 1, currentVersion: 16 }],
  capabilities: ["catalog_read"],
};

export const stubSaveSession: SaveSession = {
  saveSessionID: "session-1",
  platform: "pc",
  format: "sl2_v2",
  unsavedChanges: false,
};

export function makePort(overrides: Partial<ApplicationInfoPort> = {}): ApplicationInfoPort {
  return {
    getApplicationInfo: () => Promise.resolve(stubApplicationInfo),
    ...overrides,
  };
}

export function makeSaveSessionPort(overrides: Partial<SaveSessionPort> = {}): SaveSessionPort {
  return {
    loadSave: () => Promise.resolve(stubSaveSession),
    getLoadedSave: () => Promise.resolve(stubSaveSession),
    closeSave: () => Promise.resolve(),
    ...overrides,
  };
}

export const failingPort: ApplicationInfoPort = {
  getApplicationInfo: () => Promise.reject(new Error("bridge_call_failed")),
};

/**
 * The component-test client. `gcTime: 0` keeps rendered views from leaking
 * between tests; a test that asserts on cache contents passes its own client.
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
}

export function TestProviders({
  children,
  queryClient,
  port,
  saveSessionPort,
}: {
  children: ReactNode;
  queryClient: QueryClient;
  port?: ApplicationInfoPort;
  saveSessionPort?: SaveSessionPort;
}) {
  return (
    <QueryClientProvider client={queryClient}>
      <ApplicationInfoPortProvider port={port ?? makePort()}>
        <SaveSessionPortProvider port={saveSessionPort ?? makeSaveSessionPort()}>
          {children}
        </SaveSessionPortProvider>
      </ApplicationInfoPortProvider>
    </QueryClientProvider>
  );
}

export async function renderApp(
  ui: ReactNode,
  options: {
    port?: ApplicationInfoPort;
    saveSessionPort?: SaveSessionPort;
    locale?: Locale;
    queryClient?: QueryClient;
  } = {},
): Promise<RenderResult> {
  await activateLocale(options.locale ?? "en");

  return render(
    <I18nProvider i18n={i18n}>
      <TestProviders
        queryClient={options.queryClient ?? createTestQueryClient()}
        port={options.port}
        saveSessionPort={options.saveSessionPort}
      >
        {ui}
      </TestProviders>
    </I18nProvider>,
  );
}
