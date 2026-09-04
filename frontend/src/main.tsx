import { I18nProvider } from "@lingui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { AboutPortProvider } from "./application/about/aboutClient";
import { AppearancePortProvider } from "./application/appearance/appearanceClient";
import { ApplicationInfoPortProvider } from "./application/application-info/applicationInfoClient";
import { CatalogPortProvider } from "./application/catalog/catalogClient";
import { CharacterPortProvider } from "./application/character/characterClient";
import { DeploymentPortProvider } from "./application/deployment/deploymentClient";
import { DiagnosticsPortProvider } from "./application/diagnostics/diagnosticsClient";
import { EquipmentPortProvider } from "./application/equipment/equipmentClient";
import { FavoritesPortProvider } from "./application/favorites/favoritesClient";
import { ItemsPortProvider } from "./application/items/itemsClient";
import { ItemPreferencesProvider } from "./application/preferences/itemPreferences";
import { NetworkPortProvider } from "./application/network/networkClient";
import { SaveSessionPortProvider } from "./application/save-session/saveSessionClient";
import { SettingsPortProvider } from "./application/settings/settingsClient";
import { TemplatePortProvider } from "./application/templates/templateClient";
import { WorldPortProvider } from "./application/world/worldClient";
import { activateLocale, defaultLocale, i18n } from "./i18n/i18n";
import { wailsDesktopBridge } from "./infrastructure/bridge/desktopBridge";
import "./ui/tokens/global.css";

// Composition root: the only place that picks a concrete port implementation.
// One adapter fulfils every port, injected through React context so nothing
// below reaches it as a global.
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
        {/* The catalog provider sits beside the application-info one and above,
            not inside, the save session: Item Database reads the catalog with no
            save loaded. */}
        <ApplicationInfoPortProvider port={wailsDesktopBridge}>
          {/* The settings port and the presentational item preferences both sit
              above the save session: the Safety Profile and the item favourites
              are host state and survive closing a save. */}
          <SettingsPortProvider port={wailsDesktopBridge}>
            <ItemPreferencesProvider>
              <CatalogPortProvider port={wailsDesktopBridge}>
                <AppearancePortProvider port={wailsDesktopBridge}>
                  <FavoritesPortProvider port={wailsDesktopBridge}>
                    <SaveSessionPortProvider port={wailsDesktopBridge}>
                      <CharacterPortProvider port={wailsDesktopBridge}>
                        <DiagnosticsPortProvider port={wailsDesktopBridge}>
                          <ItemsPortProvider port={wailsDesktopBridge}>
                            <EquipmentPortProvider port={wailsDesktopBridge}>
                              <WorldPortProvider port={wailsDesktopBridge}>
                                <NetworkPortProvider port={wailsDesktopBridge}>
                                  {/* Templates, Deployment and About are Tools
                                      modules. Their ports sit here, above the
                                      screen that uses them, so nothing below
                                      reaches the adapter as a global. */}
                                  <TemplatePortProvider port={wailsDesktopBridge}>
                                    <DeploymentPortProvider port={wailsDesktopBridge}>
                                      <AboutPortProvider port={wailsDesktopBridge}>
                                        <App />
                                      </AboutPortProvider>
                                    </DeploymentPortProvider>
                                  </TemplatePortProvider>
                                </NetworkPortProvider>
                              </WorldPortProvider>
                            </EquipmentPortProvider>
                          </ItemsPortProvider>
                        </DiagnosticsPortProvider>
                      </CharacterPortProvider>
                    </SaveSessionPortProvider>
                  </FavoritesPortProvider>
                </AppearancePortProvider>
              </CatalogPortProvider>
            </ItemPreferencesProvider>
          </SettingsPortProvider>
        </ApplicationInfoPortProvider>
      </QueryClientProvider>
    </I18nProvider>
  </StrictMode>,
);
