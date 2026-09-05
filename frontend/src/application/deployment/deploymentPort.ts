/**
 * The port Deployment and Save Manager share.
 *
 * Both screens read the same targets and the same backups, because the backend
 * has one model of each. The frontend performs no file operation, no SSH
 * connection and no network request of its own: every entry below is a backend
 * call.
 */

/** One configured target. The SSH key is a path the user stated, never content. */
export type DeploymentTarget = {
  id: string;
  name: string;
  kind: string;
  savePath: string;
  startCommand?: string | undefined;
  stopCommand?: string | undefined;
  /**
   * The command that states whether the game runs. Its contract is the exit
   * code: 0 running, 1 stopped, anything else unknown.
   */
  statusCommand?: string | undefined;
  host?: string | undefined;
  port?: number | undefined;
  user?: string | undefined;
  keyPath?: string | undefined;
  /** Whether an SSH host key fingerprint was approved for this target. */
  hostKeyTrusted: boolean;
  hostKeyFingerprint?: string | undefined;
  /**
   * Whether the backend can move a save to and from this target in this build.
   * A false value disables the operations instead of offering an action that
   * must fail.
   */
  transferSupported: boolean;
  unsupportedReason?: string | undefined;
};

export type DeploymentTargets = {
  targets: readonly DeploymentTarget[];
  availableKinds: readonly string[];
};

/** The complete configuration of one target as the form states it. */
export type DeploymentTargetInput = {
  targetID?: string | undefined;
  name: string;
  kind: string;
  savePath: string;
  startCommand?: string | undefined;
  stopCommand?: string | undefined;
  statusCommand?: string | undefined;
  host?: string | undefined;
  port?: number | undefined;
  user?: string | undefined;
  keyPath?: string | undefined;
};

export type TargetTestResult = {
  targetID: string;
  reachable: boolean;
  hostKeyTrusted: boolean;
  /** "running", "stopped" or "unknown", carried verbatim. */
  gameStatus: string;
  saveExists: boolean;
  /**
   * The handshake presented a host key this configuration has never approved.
   * The connection was refused and `observedFingerprint` is what the host
   * actually presented; approving it is an explicit user decision.
   */
  hostKeyPending: boolean;
  /** The host presented a different key than the approved one. Nothing may be approved from here. */
  hostKeyChanged: boolean;
  observedFingerprint?: string | undefined;
};

export type CommandOutcome = {
  configured: boolean;
  executed: boolean;
  exitCode: number;
  detail?: string | undefined;
};

export type OperationStage = {
  stage: string;
  completed: boolean;
  detail?: string | undefined;
};

/**
 * The outcome of one long operation.
 *
 * `blocked` is a backend code the interface answers with an explicit user
 * decision: "game_running", "game_status_unknown",
 * "remote_backup_confirmation_required", "stop_game_confirmation_required" or
 * "cancelled". `targetState`, not `blocked`, is authoritative about whether
 * the irreversible replacement point was crossed, and it has three answers
 * rather than two: "replacement_undetermined" means the replacement was
 * requested and its result was never established, which is neither "the target
 * is unchanged" nor "the target was replaced".
 */
export type DeploymentOperationResult = {
  operationID: string;
  targetID: string;
  completed: boolean;
  blocked?: string | undefined;
  /** Stable failure code for an operation that stopped after it had started. */
  failure?: string | undefined;
  /** The backend's authoritative outcome of the target replacement. */
  targetState:
    | "unchanged"
    | "replaced_verified"
    | "replaced_unverified"
    | "replacement_undetermined";
  gameStatus: string;
  stages: readonly OperationStage[];
  backupID?: string | undefined;
  /** The staging file a completed download produced. */
  localPath?: string | undefined;
  stop?: CommandOutcome | undefined;
  launch?: CommandOutcome | undefined;
};

/** The explicit decisions a blocked operation is retried with. */
export type DeploymentConfirmations = {
  continueWithUnknownGameStatus?: boolean | undefined;
  confirmRemoteBackup?: boolean | undefined;
  confirmStopGame?: boolean | undefined;
};

export type DeployRequest = DeploymentConfirmations & {
  operationID: string;
  targetID: string;
  saveSessionID: string;
  expectedRevision: string;
  validationToken: string;
  confirmWarnings?: boolean | undefined;
  confirmBanRisk?: boolean | undefined;
  launchAfter?: boolean | undefined;
};

export type DownloadRequest = DeploymentConfirmations & {
  operationID: string;
  targetID: string;
  closeGameFirst?: boolean | undefined;
};

/** One backup of a target. There is deliberately no size: the table shows none. */
export type TargetBackup = {
  id: string;
  targetID: string;
  fileName: string;
  createdAt: string;
  manual: boolean;
  active: boolean;
  tags: readonly string[];
  description?: string | undefined;
};

export type TargetBackups = {
  targetID: string;
  backups: readonly TargetBackup[];
  transferSupported: boolean;
  unsupportedReason?: string | undefined;
};

/** One live report of a running operation, as the backend emits it. */
export type DeploymentProgress = {
  operationID: string;
  targetID: string;
  stage: string;
  percent: number;
  elapsedMS: number;
  finished: boolean;
};

export type DeploymentPort = {
  /**
   * Subscribes to the progress of running operations and returns the
   * unsubscribe function. Nothing here polls a target: the backend emits only
   * while an operation the user started is running.
   */
  subscribeDeploymentProgress: (listener: (progress: DeploymentProgress) => void) => () => void;
  getDeploymentTargets: () => Promise<DeploymentTargets>;
  createDeploymentTarget: (input: DeploymentTargetInput) => Promise<DeploymentTargets>;
  updateDeploymentTarget: (input: DeploymentTargetInput) => Promise<DeploymentTargets>;
  deleteDeploymentTarget: (targetID: string) => Promise<DeploymentTargets>;
  testDeploymentTarget: (targetID: string) => Promise<TargetTestResult>;
  /**
   * Approves the fingerprint the last handshake with this target presented.
   * The backend accepts only that value, so this call cannot approve an
   * invented key or a key belonging to another host.
   */
  trustDeploymentHostKey: (request: {
    targetID: string;
    fingerprint: string;
  }) => Promise<DeploymentTargets>;
  forgetDeploymentHostKey: (targetID: string) => Promise<DeploymentTargets>;

  getDeploymentGameStatus: (targetID: string) => Promise<string>;
  launchTargetGame: (targetID: string) => Promise<CommandOutcome>;
  closeTargetGame: (targetID: string) => Promise<CommandOutcome>;
  deployToTarget: (request: DeployRequest) => Promise<DeploymentOperationResult>;
  downloadFromTarget: (request: DownloadRequest) => Promise<DeploymentOperationResult>;
  /** Cooperatively cancels one running operation. Cancelling a finished one is not an error. */
  cancelDeploymentOperation: (operationID: string) => Promise<void>;

  getTargetBackups: (targetID: string) => Promise<TargetBackups>;
  createTargetBackup: (request: {
    targetID: string;
    tags: readonly string[];
    description: string;
  }) => Promise<TargetBackups>;
  activateTargetBackup: (request: {
    operationID: string;
    targetID: string;
    backupID: string;
  } & DeploymentConfirmations) => Promise<{
    operation: DeploymentOperationResult;
    backups: TargetBackups;
  }>;
  clearActiveTargetBackup: (targetID: string) => Promise<TargetBackups>;
  updateTargetBackup: (request: {
    targetID: string;
    backupID: string;
    tags: readonly string[];
    description: string;
  }) => Promise<TargetBackups>;
  deleteTargetBackup: (request: { targetID: string; backupID: string }) => Promise<TargetBackups>;
  /**
   * Opens the native Save As dialog and copies the backup to the path the user
   * agreed to. Cancelling writes nothing and returns an undefined target.
   */
  downloadTargetBackup: (request: {
    targetID: string;
    backupID: string;
  }) => Promise<{ target?: string | undefined }>;
};
