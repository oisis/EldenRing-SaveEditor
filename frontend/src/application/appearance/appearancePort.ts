import type { MutationReceipt } from "../save-session/saveSessionPort";

export type AppearancePresetSummary = {
  id: string;
  name: string;
  image: string;
  bodyType: string;
  tags: readonly string[];
};

export type GetAppearancePresetsRequest = {
  search?: string;
  tags?: readonly string[];
};

export type ApplyAppearancePresetInput = {
  saveSessionID: string;
  characterID: number;
  presetID: string;
  expectedRevision: string;
};

export type AppearancePort = {
  getAppearancePresets: (
    request: GetAppearancePresetsRequest,
  ) => Promise<readonly AppearancePresetSummary[]>;
  applyAppearancePreset: (input: ApplyAppearancePresetInput) => Promise<MutationReceipt>;
};
