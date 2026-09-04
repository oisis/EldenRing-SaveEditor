import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useDeploymentPort } from "./deploymentClient";
import type {
  CommandOutcome,
  DeploymentOperationResult,
  DeploymentTargetInput,
  DeploymentTargets,
  DeployRequest,
  DownloadRequest,
  TargetBackups,
  TargetTestResult,
} from "./deploymentPort";

/** The configured deployment targets. They are host configuration, not save state. */
export function useDeploymentTargets() {
  const port = useDeploymentPort();

  return useQuery<DeploymentTargets>({
    queryKey: queryKeys.deploymentTargets(),
    queryFn: () => port.getDeploymentTargets(),
    retry: false,
  });
}

/**
 * The confirmed game state of one target.
 *
 * It is never polled: section 2 of the deployment specification states the
 * application does not contact a target in the background, so the query has no
 * interval, no refetch on focus and no retry. It refreshes when the user asks.
 */
export function useDeploymentGameStatus(targetID: string | undefined) {
  const port = useDeploymentPort();

  return useQuery<string>({
    queryKey: queryKeys.deploymentGameStatus(targetID ?? ""),
    queryFn: () => port.getDeploymentGameStatus(targetID as string),
    enabled: targetID !== undefined,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

/** The backup library of one target, shared by Deployment and Save Manager. */
export function useTargetBackups(targetID: string | undefined) {
  const port = useDeploymentPort();

  return useQuery<TargetBackups>({
    queryKey: queryKeys.targetBackups(targetID ?? ""),
    queryFn: () => port.getTargetBackups(targetID as string),
    enabled: targetID !== undefined,
    retry: false,
  });
}

/**
 * The target configuration mutations.
 *
 * Every one of them answers with the whole library the backend now holds, so
 * the cache is set from the backend's own answer rather than patched locally.
 * None of them is a save mutation: no revision moves and no receipt exists, so
 * nothing here touches a session key.
 */
export function useDeploymentTargetMutations() {
  const port = useDeploymentPort();
  const queryClient = useQueryClient();
  const accept = (targets: DeploymentTargets) =>
    queryClient.setQueryData(queryKeys.deploymentTargets(), targets);

  const create = useMutation<DeploymentTargets, Error, DeploymentTargetInput>({
    mutationFn: (input) => port.createDeploymentTarget(input),
    onSuccess: accept,
  });
  const update = useMutation<DeploymentTargets, Error, DeploymentTargetInput>({
    mutationFn: (input) => port.updateDeploymentTarget(input),
    onSuccess: accept,
  });
  const remove = useMutation<DeploymentTargets, Error, string>({
    mutationFn: (targetID) => port.deleteDeploymentTarget(targetID),
    onSuccess: accept,
  });
  const forgetHostKey = useMutation<DeploymentTargets, Error, string>({
    mutationFn: (targetID) => port.forgetDeploymentHostKey(targetID),
    onSuccess: accept,
  });
  const test = useMutation<TargetTestResult, Error, string>({
    mutationFn: (targetID) => port.testDeploymentTarget(targetID),
    onSuccess: (result) =>
      queryClient.setQueryData(queryKeys.deploymentGameStatus(result.targetID), result.gameStatus),
  });

  return { create, update, remove, forgetHostKey, test };
}

/**
 * The deployment operations.
 *
 * The game-status key is refreshed after every one of them, because launching,
 * stopping or replacing a save is exactly what can change it. The backup
 * library of the target is refreshed too: a deployment that replaced an
 * existing save created a mandatory backup.
 */
export function useDeploymentOperations() {
  const port = useDeploymentPort();
  const queryClient = useQueryClient();
  const refresh = async (targetID: string) => {
    await queryClient.invalidateQueries({ queryKey: queryKeys.deploymentGameStatus(targetID) });
    await queryClient.invalidateQueries({ queryKey: queryKeys.targetBackups(targetID) });
  };

  const launch = useMutation<CommandOutcome, Error, string>({
    mutationFn: (targetID) => port.launchTargetGame(targetID),
    onSuccess: (_outcome, targetID) => refresh(targetID),
  });
  const closeGame = useMutation<CommandOutcome, Error, string>({
    mutationFn: (targetID) => port.closeTargetGame(targetID),
    onSuccess: (_outcome, targetID) => refresh(targetID),
  });
  const deploy = useMutation<DeploymentOperationResult, Error, DeployRequest>({
    mutationFn: (request) => port.deployToTarget(request),
    onSuccess: (result) => refresh(result.targetID),
  });
  const download = useMutation<DeploymentOperationResult, Error, DownloadRequest>({
    mutationFn: (request) => port.downloadFromTarget(request),
    onSuccess: (result) => refresh(result.targetID),
  });
  const cancel = useMutation<void, Error, string>({
    mutationFn: (operationID) => port.cancelDeploymentOperation(operationID),
  });

  return { launch, closeGame, deploy, download, cancel };
}

/**
 * The Save Manager mutations. They act on the same backups the deployment
 * operations create, so they share one cache key with them.
 */
export function useTargetBackupMutations() {
  const port = useDeploymentPort();
  const queryClient = useQueryClient();
  const accept = (backups: TargetBackups) =>
    queryClient.setQueryData(queryKeys.targetBackups(backups.targetID), backups);

  const create = useMutation<
    TargetBackups,
    Error,
    { targetID: string; tags: readonly string[]; description: string }
  >({
    mutationFn: (request) => port.createTargetBackup(request),
    onSuccess: accept,
  });
  const activate = useMutation<
    { operation: DeploymentOperationResult; backups: TargetBackups },
    Error,
    {
      operationID: string;
      targetID: string;
      backupID: string;
      continueWithUnknownGameStatus?: boolean | undefined;
      confirmRemoteBackup?: boolean | undefined;
    }
  >({
    mutationFn: (request) => port.activateTargetBackup(request),
    onSuccess: async ({ backups, operation }) => {
      accept(backups);
      await queryClient.invalidateQueries({
        queryKey: queryKeys.deploymentGameStatus(operation.targetID),
      });
    },
  });
  const clearActive = useMutation<TargetBackups, Error, string>({
    mutationFn: (targetID) => port.clearActiveTargetBackup(targetID),
    onSuccess: accept,
  });
  const update = useMutation<
    TargetBackups,
    Error,
    { targetID: string; backupID: string; tags: readonly string[]; description: string }
  >({
    mutationFn: (request) => port.updateTargetBackup(request),
    onSuccess: accept,
  });
  const remove = useMutation<TargetBackups, Error, { targetID: string; backupID: string }>({
    mutationFn: (request) => port.deleteTargetBackup(request),
    onSuccess: accept,
  });
  const download = useMutation<
    { target?: string | undefined },
    Error,
    { targetID: string; backupID: string }
  >({
    mutationFn: (request) => port.downloadTargetBackup(request),
  });

  return { create, activate, clearActive, update, remove, download };
}
