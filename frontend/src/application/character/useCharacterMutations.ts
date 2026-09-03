import { useCallback, useRef, useState } from "react";
import { useAppearancePort } from "../appearance/appearanceClient";
import type { ApplyAppearancePresetInput } from "../appearance/appearancePort";
import { type AppError, toAppError } from "../errors/appError";
import { useFavoritesPort } from "../favorites/favoritesClient";
import type {
  ApplyFavoritePresetInput,
  DeleteFavoritePresetInput,
  SetFavoritePresetInput,
} from "../favorites/favoritesPort";
import type { MutationReceipt } from "../save-session/saveSessionPort";
import { useCharacterPort } from "./characterClient";
import type {
  SetCharacterGenderInput,
  SetCharacterNameInput,
  SetCharacterStartingClassInput,
  SetCharacterStatsInput,
} from "./characterPort";

export type CharacterMutations = {
  isBusy: boolean;
  error: AppError | undefined;
  clearError: () => void;
  setName: (input: SetCharacterNameInput) => Promise<boolean>;
  setStats: (input: SetCharacterStatsInput) => Promise<boolean>;
  setStartingClass: (input: SetCharacterStartingClassInput) => Promise<boolean>;
  setGender: (input: SetCharacterGenderInput) => Promise<boolean>;
  applyAppearancePreset: (input: ApplyAppearancePresetInput) => Promise<boolean>;
  setFavoritePreset: (input: SetFavoritePresetInput) => Promise<boolean>;
  applyFavoritePreset: (input: ApplyFavoritePresetInput) => Promise<boolean>;
  deleteFavoritePreset: (input: DeleteFavoritePresetInput) => Promise<boolean>;
};

export function useCharacterMutations(
  applyReceipt: (receipt: MutationReceipt) => Promise<unknown>,
): CharacterMutations {
  const characterPort = useCharacterPort();
  const appearancePort = useAppearancePort();
  const favoritesPort = useFavoritesPort();

  const running = useRef(false);
  const [isBusy, setBusy] = useState(false);
  const [error, setError] = useState<AppError | undefined>(undefined);

  const run = useCallback(
    async (call: () => Promise<MutationReceipt>): Promise<boolean> => {
      if (running.current) return false;
      running.current = true;
      setBusy(true);
      setError(undefined);
      try {
        await applyReceipt(await call());
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
    setName: (input) => run(() => characterPort.setCharacterName(input)),
    setStats: (input) => run(() => characterPort.setCharacterStats(input)),
    setStartingClass: (input) => run(() => characterPort.setCharacterStartingClass(input)),
    setGender: (input) => run(() => characterPort.setCharacterGender(input)),
    applyAppearancePreset: (input) => run(() => appearancePort.applyAppearancePreset(input)),
    setFavoritePreset: (input) => run(() => favoritesPort.setFavoritePreset(input)),
    applyFavoritePreset: (input) => run(() => favoritesPort.applyFavoritePreset(input)),
    deleteFavoritePreset: (input) => run(() => favoritesPort.deleteFavoritePreset(input)),
  };
}
