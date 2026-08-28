// The ONLY module allowed to import the generated Wails bindings. Everything
// above this layer depends on the application port, never on generated code or
// on `window.go`.

import { GetApplicationInfo } from "../../../wailsjs/go/desktop/Bridge";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../../application/application-info/applicationInfoPort";

/**
 * The stable code a failed bridge call is reported with. The transport error is
 * deliberately dropped here: a Wails rejection can carry a Go error string or a
 * runtime stack, and neither may reach the interface. The UI maps this code to
 * a localized message and never renders the code itself.
 */
export const bridgeFailureCode = "bridge_call_failed";

export const wailsApplicationInfoBridge: ApplicationInfoPort = {
  getApplicationInfo: async (): Promise<ApplicationInfo> => {
    let result: Awaited<ReturnType<typeof GetApplicationInfo>>;
    try {
      result = await GetApplicationInfo();
    } catch {
      throw new Error(bridgeFailureCode);
    }

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
};
