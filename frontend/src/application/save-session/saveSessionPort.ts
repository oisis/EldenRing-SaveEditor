import type { ChangedScope } from "../changedScopes";

/**
 * The port the application layer needs in order to work with a save session.
 * Infrastructure implements it; feature modules depend on it through the hooks
 * in this directory and never on the transport that fulfils it.
 *
 * `SaveSession` is a projection of the backend session contract as it exists
 * today and adds nothing to it. Every field is carried verbatim: the frontend
 * invents no revision, derives no source and computes no change state.
 */
export type SaveSession = {
  /** Backend session identifier, carried verbatim; the UI does not interpret it. */
  saveSessionID: string;
  /** Backend platform identifier, carried verbatim and never normalised. */
  platform: string;
  /** Backend save format identifier, carried verbatim and never normalised. */
  format: string;
  /**
   * The exact path the backend created the session snapshot from, carried
   * verbatim. It is display metadata: the frontend never resolves it, never
   * reads it and never rebuilds a path from it.
   */
  sourcePath: string;
  /** `local` or `temporary`, exactly as the backend reports it. */
  sourceKind: string;
  /**
   * The backend's canonical decimal revision of this session. It stays a string
   * end to end: the frontend never parses, increments or compares it
   * numerically.
   */
  saveRevision: string;
  /** The backend's change state. The frontend never derives or overrides it. */
  unsavedChanges: boolean;
  /**
   * The backend's canonical decimal position of this session's `session.changed`
   * stream, `"0"` for a session that has committed nothing. It is the baseline a
   * listener resynchronises against and, like `saveRevision`, stays a string end
   * to end: it is never parsed into a number, incremented or rounded.
   */
  eventSequence: string;
};

/**
 * One committed backend mutation of a save session, exactly as the backend
 * publishes it. It is a notification and never a source of state: a listener
 * refreshes the getters named by `changedScopes` and reconstructs nothing from
 * the event itself.
 */
export type SessionChangedEvent = {
  /** Monotonic position in this session's event stream, canonical decimal. */
  sequence: string;
  /** The single execution that committed. */
  operationID: string;
  /** Stable kind of the mutation: the EndpointID that initiated it. */
  operationKind: string;
  saveSessionID: string;
  /** The revision the mutation created, carried verbatim. */
  saveRevision: string;
  /** Exactly the scopes of the committing mutation's receipt. */
  changedScopes: readonly ChangedScope[];
};

/**
 * What a source file is, as the backend contract defines it. The two values are
 * the only accepted ones and are sent exactly as written here: the frontend adds
 * no alias, no empty form and no default, so a caller must always state one.
 */
export type SaveSourceKind = "local" | "temporary";

export type SaveSessionPort = {
  /**
   * Opens the host's native file dialog and resolves with the chosen path.
   *
   * Cancelling is an ordinary outcome and not an error: it resolves with an
   * empty string, and the caller must not load anything for it. The returned
   * path is passed on to `loadSave` unchanged — the frontend never trims,
   * resolves or validates it, because recognising a save is the backend's
   * contract.
   *
   * It lives on this port rather than on one of its own because choosing the
   * file is the first step of the session flow and its only caller; a separate
   * host port would be a second injection point for a single method.
   */
  selectSaveFile: () => Promise<string>;
  /**
   * Creates a session from a source the host layer supplied. All three
   * arguments are passed to the backend exactly as received: the backend owns
   * path handling, platform validation and the source-kind rule, and rejects
   * anything it does not accept.
   */
  loadSave: (
    source: string,
    expectedPlatform: string,
    sourceKind: SaveSourceKind,
  ) => Promise<SaveSession>;
  getLoadedSave: (saveSessionID: string) => Promise<SaveSession>;
  closeSave: (saveSessionID: string) => Promise<void>;
  /**
   * Subscribes to committed backend mutations and returns the unsubscribe
   * function. The port carries the typed event; the host mechanism behind it
   * belongs to the infrastructure adapter alone.
   *
   * `null` is delivered for a payload the adapter could not validate against the
   * contract. It is a signal and not an event: it says one notification arrived
   * and could not be understood, so the listener resynchronises instead of
   * acting on a fragment. A rejected payload is therefore never silently
   * dropped.
   */
  subscribeSessionChanged: (listener: (event: SessionChangedEvent | null) => void) => () => void;
};
