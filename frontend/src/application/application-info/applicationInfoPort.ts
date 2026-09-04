/**
 * The port the application layer needs in order to show what the backend is.
 * Infrastructure implements it; features depend on it through the hook in
 * `useApplicationInfo.ts` and never on the transport that fulfils it.
 *
 * These types are a deliberate projection of what the foundation screen
 * renders, not a hand-maintained copy of the backend transport DTO. The backend
 * result is the single source of truth; the adapter maps it onto this shape in
 * one place, so a contract change surfaces there instead of spreading through
 * the UI.
 */
export type SchemaSupport = {
  /** Backend schema identifier, rendered verbatim; the UI does not interpret it. */
  name: string;
  minimumVersion: number;
  currentVersion: number;
};

export type ApplicationInfo = {
  version: string;
  build: string;
  platform: string;
  schemas: readonly SchemaSupport[];
  capabilities: readonly string[];
};

export type ApplicationInfoPort = {
  getApplicationInfo: () => Promise<ApplicationInfo>;
};
