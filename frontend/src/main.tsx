import { I18nProvider } from "@lingui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { ApplicationInfoPortProvider } from "./application/application-info/applicationInfoClient";
import { activateLocale, defaultLocale, i18n } from "./i18n/i18n";
import { wailsApplicationInfoBridge } from "./infrastructure/bridge/applicationInfoBridge";
import "./ui/tokens/global.css";

// Composition root: the only place that picks a concrete port implementation.
const queryClient = new QueryClient();

const container = document.getElementById("root");
if (!container) {
  throw new Error("Root container #root is missing from index.html");
}

await activateLocale(defaultLocale);

createRoot(container).render(
  <StrictMode>
    <I18nProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <ApplicationInfoPortProvider port={wailsApplicationInfoBridge}>
          <App />
        </ApplicationInfoPortProvider>
      </QueryClientProvider>
    </I18nProvider>
  </StrictMode>,
);
