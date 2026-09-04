/**
 * The port the application layer needs in order to read and change the global
 * host application settings. Infrastructure implements it; feature modules
 * depend on it through the hooks in this directory and never on the transport
 * that fulfils it.
 *
 * The Safety Profile is a backend-owned setting. The frontend may present and
 * cache the value, but it never interprets it: which limits a profile applies
 * and which resources it reveals are backend decisions, and no call carries a
 * profile as an argument.
 */

/** The active profile plus the closed vocabulary the backend accepts. */
export type SafetyProfileSettings = {
  /** The profile now in effect, carried verbatim. */
  safetyProfile: string;
  /** Every value the backend accepts, in the backend's own order. */
  availableProfiles: readonly string[];
  /** The value a host that never stored one runs under. */
  defaultProfile: string;
};

/**
 * The whole settings surface behind one provider. The Safety Profile and the
 * host settings are both backend-owned application settings, so one port
 * carries them and the Settings screen needs one provider rather than two.
 */
export type SettingsPort = HostSettingsPort & {
  getSafetyProfile: () => Promise<SafetyProfileSettings>;
  /** Stores one profile and returns the settings now in effect. */
  setSafetyProfile: (safetyProfile: string) => Promise<SafetyProfileSettings>;
};

/**
 * The persistent host settings of `Tools → Settings`.
 *
 * They are backend state, exactly like the Safety Profile: React Query caches
 * them and no component keeps a second authoritative copy.
 */
export type HostSettings = {
  /**
   * Whether Review Changes may be skipped for an operation the completed
   * validation reported as normal risk. It never skips the validation itself.
   */
  skipReviewForNormalRisk: boolean;
  /** "ask" or "always", carried verbatim. Neither value disables the backup. */
  remoteBackupPolicy: string;
  availableRemoteBackupPolicies: readonly string[];
  defaultRemoteBackupPolicy: string;
  /** Whether this host has a configuration directory that can be opened. */
  configurationDirectoryExists: boolean;
  logDirectoryExists: boolean;
};

/** The two host directories the backend can open. Identifiers, never paths. */
export type HostLocation = "configuration" | "logs";

/**
 * The result of a diagnostic report export. A cancelled Save As dialog reports
 * `exported: false`, which is an ordinary outcome and never an error.
 */
export type DiagnosticReportResult = {
  exported: boolean;
  recordCount: number;
};

export type HostSettingsPort = {
  getHostSettings: () => Promise<HostSettings>;
  setHostSettings: (settings: {
    skipReviewForNormalRisk: boolean;
    remoteBackupPolicy: string;
  }) => Promise<HostSettings>;
  /**
   * Opens one known host directory. The frontend states an identifier and never
   * a path: there is no bridge call that opens an arbitrary location.
   */
  openHostLocation: (location: HostLocation) => Promise<void>;
  /** Opens the native Save As dialog and writes the redacted report. */
  exportDiagnosticReport: (saveSessionID?: string) => Promise<DiagnosticReportResult>;
};
