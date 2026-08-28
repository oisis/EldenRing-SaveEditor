import { I18nProvider } from "@lingui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderResult, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { ApplicationInfoPortProvider } from "../application/application-info/applicationInfoClient";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../application/application-info/applicationInfoPort";
import { activateLocale, i18n, type Locale } from "../i18n/i18n";

/**
 * Components are exercised through the application port, never through a mock
 * of the generated Wails bindings: a component test must fail when the
 * application contract breaks, not when the transport detail changes.
 */
export const stubApplicationInfo: ApplicationInfo = {
  version: "2.0.0-test",
  schemas: [{ name: "game_catalog", minimumVersion: 1, currentVersion: 16 }],
  capabilities: ["catalog_read"],
};

export function makePort(overrides: Partial<ApplicationInfoPort> = {}): ApplicationInfoPort {
  return {
    getApplicationInfo: () => Promise.resolve(stubApplicationInfo),
    ...overrides,
  };
}

export const failingPort: ApplicationInfoPort = {
  getApplicationInfo: () => Promise.reject(new Error("bridge_call_failed")),
};

export async function renderApp(
  ui: ReactNode,
  options: { port?: ApplicationInfoPort; locale?: Locale } = {},
): Promise<RenderResult> {
  await activateLocale(options.locale ?? "en");

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });

  return render(
    <I18nProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <ApplicationInfoPortProvider port={options.port ?? makePort()}>
          {ui}
        </ApplicationInfoPortProvider>
      </QueryClientProvider>
    </I18nProvider>,
  );
}
