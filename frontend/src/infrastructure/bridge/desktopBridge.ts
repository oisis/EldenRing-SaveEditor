// The ONLY module allowed to import the generated Wails bindings. Everything
// above this layer depends on the application ports, never on generated code or
// on `window.go`.

import {
  CloseSave,
  GetApplicationInfo,
  GetLoadedSave,
  LoadSave,
} from "../../../wailsjs/go/desktop/Bridge";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../../application/application-info/applicationInfoPort";
import type { SaveSession, SaveSessionPort } from "../../application/save-session/saveSessionPort";

/**
 * The stable code a failed bridge call is reported with. The transport error is
 * deliberately dropped here: a Wails rejection can carry a Go error string or a
 * runtime stack, and neither may reach the interface. The UI maps this code to
 * a localized message and never renders the code itself.
 *
 * The code says that the call failed and nothing more. Classifying a domain
 * failure would mean reading the rejection text, which is exactly what this
 * boundary refuses to do; a structured backend error contract is what a finer
 * distinction has to come from.
 */
export const bridgeFailureCode = "bridge_call_failed";

async function callBridge<T>(call: () => Promise<T>): Promise<T> {
  try {
    return await call();
  } catch {
    throw new Error(bridgeFailureCode);
  }
}

/** Projects the generated session result onto the application port shape. */
function toSaveSession(result: Awaited<ReturnType<typeof GetLoadedSave>>): SaveSession {
  return {
    saveSessionID: result.saveSessionID,
    platform: result.platform,
    format: result.format,
    unsavedChanges: result.unsavedChanges,
  };
}

/**
 * The single adapter behind every application port. A second, parallel
 * adaptation layer would give the generated bindings a second way into the
 * application, so all ports are fulfilled here.
 */
export const wailsDesktopBridge: ApplicationInfoPort & SaveSessionPort = {
  getApplicationInfo: async (): Promise<ApplicationInfo> => {
    const result = await callBridge(GetApplicationInfo);

    return {
      version: result.applicationVersion,
      schemas: result.supportedSchemas.map((schema) => ({
        name: schema.name,
        minimumVersion: schema.minimumVersion,
        currentVersion: schema.currentVersion,
      })),
      capabilities: [...result.capabilities],
    };
  },

  loadSave: async (source, expectedPlatform) =>
    toSaveSession(await callBridge(() => LoadSave(source, expectedPlatform))),

  getLoadedSave: async (saveSessionID) =>
    toSaveSession(await callBridge(() => GetLoadedSave(saveSessionID))),

  closeSave: async (saveSessionID) => {
    await callBridge(() => CloseSave(saveSessionID));
  },
};
