import type { MutationReceipt } from "../save-session/saveSessionPort";

/**
 * The 22 NetworkParam values.
 * Identifiers match backend `gamecatalog.NetworkParamValues` exactly,
 * including historical `singGetMax`.
 */
export type NetworkParamValues = {
  maxBreakInTargetListCount: number;
  breakInRequestIntervalTimeSec: number;
  breakInRequestTimeOutSec: number;
  breakInRequestAreaCount: number;

  summonTimeoutTime: number;

  reloadSignIntervalTime2: number;
  reloadSignTotalCount: number;
  reloadSignCellCount: number;
  updateSignIntervalTime: number;
  singGetMax: number;
  signDownloadSpan: number;
  signUpdateSpan: number;

  reloadVisitListCoolTime: number;
  maxCoopBlueSummonCount: number;
  maxVisitListCount: number;
  reloadSearchCoopBlueMin: number;
  reloadSearchCoopBlueMax: number;
  allAreaSearchRateCoopBlue: number;
  allAreaSearchRateVsBlue: number;

  visitorListMax: number;
  visitorTimeOutTime: number;
  visitorDownloadSpan: number;
};

export type NetworkPreset = {
  id: string;
  parameters: NetworkParamValues;
};

export type NetworkSettingsSnapshot = {
  saveSessionID: string;
  saveRevision: string;
  parameters: NetworkParamValues;
};

export type NetworkPresetsResult = {
  presets: readonly NetworkPreset[];
};

export type SetNetworkSettingsResult = MutationReceipt & {
  networkSettings: NetworkParamValues;
};

export type NetworkPort = {
  getNetworkSettings(saveSessionID: string): Promise<NetworkSettingsSnapshot>;
  getNetworkPresets(presetID?: string): Promise<NetworkPresetsResult>;
  setNetworkSettings(
    saveSessionID: string,
    networkSettings: NetworkParamValues,
    expectedRevision: string,
  ): Promise<SetNetworkSettingsResult>;
};
