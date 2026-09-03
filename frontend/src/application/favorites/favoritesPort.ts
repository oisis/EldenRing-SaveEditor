import type { MutationReceipt } from "../save-session/saveSessionPort";

export type FavoritePreset = {
  favoriteSlotID: number;
  active: boolean;
};

export type SaveFavoritePresets = {
  saveSessionID: string;
  presets: readonly FavoritePreset[];
};

export type SetFavoritePresetInput = {
  saveSessionID: string;
  favoriteSlotID: number;
  sourceCharacterID: number;
  expectedRevision: string;
};

export type ApplyFavoritePresetInput = {
  saveSessionID: string;
  characterID: number;
  favoriteSlotID: number;
  expectedRevision: string;
};

export type DeleteFavoritePresetInput = {
  saveSessionID: string;
  favoriteSlotID: number;
  expectedRevision: string;
};

export type FavoritesPort = {
  getFavoritePresets: (
    saveSessionID: string,
    favoriteSlotID?: number,
  ) => Promise<SaveFavoritePresets>;
  setFavoritePreset: (input: SetFavoritePresetInput) => Promise<MutationReceipt>;
  applyFavoritePreset: (input: ApplyFavoritePresetInput) => Promise<MutationReceipt>;
  deleteFavoritePreset: (input: DeleteFavoritePresetInput) => Promise<MutationReceipt>;
};
