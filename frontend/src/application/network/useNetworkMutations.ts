import { useCallback, useRef, useState } from "react";
import { type AppError, toAppError } from "../errors/appError";
import { useNetworkPort } from "./networkClient";
import type {
  NetworkParamValues,
  SetNetworkSettingsResult,
} from "./networkPort";

export type SetNetworkSettingsInput = {
  saveSessionID: string;
  networkSettings: NetworkParamValues;
  expectedRevision: string;
};

export type NetworkMutations = {
  isBusy: boolean;
  error: AppError | undefined;
  clearError: () => void;
  setNetworkSettings: (input: SetNetworkSettingsInput) => Promise<boolean>;
};

export function useSetNetworkSettings(
  applyReceipt: (receipt: SetNetworkSettingsResult) => Promise<unknown>,
): NetworkMutations {
  const port = useNetworkPort();
  const running = useRef(false);
  const [isBusy, setBusy] = useState(false);
  const [error, setError] = useState<AppError | undefined>(undefined);

  const run = useCallback(
    async (call: () => Promise<SetNetworkSettingsResult>): Promise<boolean> => {
      if (running.current) return false;
      running.current = true;
      setBusy(true);
      setError(undefined);
      try {
        const result = await call();
        await applyReceipt(result);
        return true;
      } catch (reason) {
        setError(toAppError(reason));
        return false;
      } finally {
        running.current = false;
        setBusy(false);
      }
    },
    [applyReceipt],
  );

  return {
    isBusy,
    error,
    clearError: () => setError(undefined),
    setNetworkSettings: (input) =>
      run(() =>
        port.setNetworkSettings(
          input.saveSessionID,
          input.networkSettings,
          input.expectedRevision,
        ),
      ),
  };
}
