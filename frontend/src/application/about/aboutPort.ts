/**
 * The port `About & Updates` needs. The backend owns every address and the
 * whole update check: the frontend can ask for an approved link by identifier
 * and can ask for one manual check, and it can do nothing else.
 */

/** One approved project address, reported by the backend allowlist. */
export type ProjectLink = {
  /** A backend identifier such as "repository"; the frontend never invents one. */
  id: string;
  url: string;
};

/**
 * The outcome of one manual update check.
 *
 * `status` is a backend code: "current", "available", "unknown" or
 * "unavailable". The frontend words it and never derives a different answer
 * from the two version strings itself.
 */
export type UpdateCheck = {
  status: string;
  currentVersion: string;
  latestVersion?: string | undefined;
  releaseURL?: string | undefined;
  publishedAt?: string | undefined;
  comparisonPossible: boolean;
};

export type AboutPort = {
  getProjectLinks: () => Promise<readonly ProjectLink[]>;
  /** Opens one approved address in the host's default browser. */
  openProjectLink: (linkID: string) => Promise<void>;
  /** Performs exactly one check. Nothing schedules this call. */
  checkForUpdates: () => Promise<UpdateCheck>;
};
