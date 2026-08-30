/**
 * The port the application layer needs in order to work with a save session.
 * Infrastructure implements it; feature modules depend on it through the hooks
 * in this directory and never on the transport that fulfils it.
 *
 * `SaveSession` is a projection of the backend session contract as it exists
 * today: the backend reports exactly these four fields and the frontend adds
 * nothing to them. No revision, no source path and no validation status are
 * invented here; when the backend contract grows, this type grows with it.
 */
export type SaveSession = {
  /** Backend session identifier, carried verbatim; the UI does not interpret it. */
  saveSessionID: string;
  /** Backend platform identifier, carried verbatim and never normalised. */
  platform: string;
  /** Backend save format identifier, carried verbatim and never normalised. */
  format: string;
  unsavedChanges: boolean;
};

export type SaveSessionPort = {
  /**
   * Creates a session from a source the host layer supplies. Both arguments are
   * passed to the backend exactly as received: the backend owns path handling
   * and platform validation.
   */
  loadSave: (source: string, expectedPlatform: string) => Promise<SaveSession>;
  getLoadedSave: (saveSessionID: string) => Promise<SaveSession>;
  closeSave: (saveSessionID: string) => Promise<void>;
};
