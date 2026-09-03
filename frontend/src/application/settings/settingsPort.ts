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

export type SettingsPort = {
  getSafetyProfile: () => Promise<SafetyProfileSettings>;
  /** Stores one profile and returns the settings now in effect. */
  setSafetyProfile: (safetyProfile: string) => Promise<SafetyProfileSettings>;
};
