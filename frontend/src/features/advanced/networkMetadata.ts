import { msg } from "@lingui/core/macro";
import type { MessageDescriptor } from "@lingui/core";
import type { NetworkParamValues } from "../../application/network/networkPort";

export type ParamMetadata = {
  key: keyof NetworkParamValues;
  label: MessageDescriptor;
  description: MessageDescriptor;
  min: number;
  max: number;
  step: number;
  unit: string;
  isInteger?: boolean;
};

export type NetworkGroupDefinition = {
  id: string;
  title: MessageDescriptor;
  description: MessageDescriptor;
  fields: readonly ParamMetadata[];
};

export type PresetRoleDefinition = {
  id: string;
  title: MessageDescriptor;
  fasterPresetID: string;
  aggressivePresetID: string;
};

export const presetRoles: readonly PresetRoleDefinition[] = [
  {
    id: "reds",
    title: msg`Reds / Invader`,
    fasterPresetID: "faster-reds",
    aggressivePresetID: "aggressive-reds",
  },
  {
    id: "summons",
    title: msg`Summon signs`,
    fasterPresetID: "faster-summons",
    aggressivePresetID: "aggressive-summons",
  },
  {
    id: "blue",
    title: msg`Blue matchmaking`,
    fasterPresetID: "faster-blue",
    aggressivePresetID: "aggressive-blue",
  },
  {
    id: "summonHost",
    title: msg`Summon host`,
    fasterPresetID: "faster-summon-host",
    aggressivePresetID: "aggressive-summon-host",
  },
  {
    id: "summonGuest",
    title: msg`Summon guest`,
    fasterPresetID: "faster-summon-guest",
    aggressivePresetID: "aggressive-summon-guest",
  },
  {
    id: "hunter",
    title: msg`Hunter`,
    fasterPresetID: "faster-hunter",
    aggressivePresetID: "aggressive-hunter",
  },
];

export const networkGroups: readonly NetworkGroupDefinition[] = [
  {
    id: "invader",
    title: msg`Invader`,
    description: msg`Matchmaking parameters for Bloody / Recusant Finger invasions.`,
    fields: [
      {
        key: "maxBreakInTargetListCount",
        label: msg`Max Targets`,
        description: msg`Invasion candidate list size polled per matchmaking cycle.`,
        min: 1,
        max: 20,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "breakInRequestAreaCount",
        label: msg`Search Areas`,
        description: msg`Number of break-in areas queried per invasion search cycle.`,
        min: 1,
        max: 50,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "breakInRequestIntervalTimeSec",
        label: msg`Request Interval`,
        description: msg`Delay between matchmaking search retries.`,
        min: 2,
        max: 30,
        step: 0.5,
        unit: "s",
        isInteger: false,
      },
      {
        key: "breakInRequestTimeOutSec",
        label: msg`Request Timeout`,
        description: msg`Timeout before a single matchmaking attempt is abandoned.`,
        min: 3,
        max: 20,
        step: 0.5,
        unit: "s",
        isInteger: false,
      },
    ],
  },
  {
    id: "summonHost",
    title: msg`Summon host`,
    description: msg`Visibility, refresh rate and timeouts when summoning cooperators.`,
    fields: [
      {
        key: "summonTimeoutTime",
        label: msg`Summon Timeout`,
        description: msg`Seconds before a pending summon request expires.`,
        min: 1,
        max: 999,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "reloadSignIntervalTime2",
        label: msg`Sign Refresh Interval`,
        description: msg`How frequently the game refreshes the summon sign list.`,
        min: 1,
        max: 1000,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "reloadSignTotalCount",
        label: msg`Signs Total Count`,
        description: msg`Maximum total signs downloaded per refresh cycle.`,
        min: 1,
        max: 128,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "reloadSignCellCount",
        label: msg`Signs Per Cell`,
        description: msg`Maximum signs downloaded per world cell.`,
        min: 1,
        max: 99,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "singGetMax",
        label: msg`Sign Get Max`,
        description: msg`Hard cap on signs retrievable from the server.`,
        min: 1,
        max: 128,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "signDownloadSpan",
        label: msg`Sign Download Span`,
        description: msg`Download span interval for sign requests.`,
        min: 1,
        max: 1000,
        step: 1,
        unit: "s",
        isInteger: false,
      },
    ],
  },
  {
    id: "summonGuest",
    title: msg`Summon guest`,
    description: msg`Upload and broadcast intervals when placing a summon sign.`,
    fields: [
      {
        key: "updateSignIntervalTime",
        label: msg`Sign Upload Interval`,
        description: msg`How frequently your placed summon sign is re-uploaded to the server.`,
        min: 1,
        max: 1000,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "signUpdateSpan",
        label: msg`Sign Update Span`,
        description: msg`Span interval for sign broadcast updates.`,
        min: 1,
        max: 1000,
        step: 1,
        unit: "s",
        isInteger: false,
      },
    ],
  },
  {
    id: "hunter",
    title: msg`Hunter / Blue matchmaking`,
    description: msg`Matchmaking intervals and search rates for Blue Cipher Ring hunters.`,
    fields: [
      {
        key: "reloadVisitListCoolTime",
        label: msg`Search Cooldown`,
        description: msg`Cooldown between Blue Cipher Ring host search attempts.`,
        min: 1,
        max: 1000,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "maxCoopBlueSummonCount",
        label: msg`Max Blue Summons`,
        description: msg`Maximum number of blue cooperators allowed.`,
        min: 1,
        max: 10,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "maxVisitListCount",
        label: msg`Visit List Size`,
        description: msg`Candidate list capacity for hunter matchmaking targets.`,
        min: 1,
        max: 50,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "reloadSearchCoopBlueMin",
        label: msg`Reload Search Min`,
        description: msg`Minimum delay for co-op blue reload search.`,
        min: 1,
        max: 999,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "reloadSearchCoopBlueMax",
        label: msg`Reload Search Max`,
        description: msg`Maximum delay for co-op blue reload search.`,
        min: 1,
        max: 999,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "allAreaSearchRateCoopBlue",
        label: msg`All Area Search Rate (Co-op)`,
        description: msg`Percentage rate for searching all areas for co-op blue summons.`,
        min: 0,
        max: 100,
        step: 1,
        unit: "%",
        isInteger: true,
      },
      {
        key: "allAreaSearchRateVsBlue",
        label: msg`All Area Search Rate (Vs Blue)`,
        description: msg`Percentage rate for searching all areas vs blue summons.`,
        min: 0,
        max: 100,
        step: 1,
        unit: "%",
        isInteger: true,
      },
    ],
  },
  {
    id: "visitor",
    title: msg`Visitor / additional parameters`,
    description: msg`Visitor list capacities, timeouts and download spans.`,
    fields: [
      {
        key: "visitorListMax",
        label: msg`Visitor List Max`,
        description: msg`Maximum visitor entries allowed in the list.`,
        min: 1,
        max: 100,
        step: 1,
        unit: "",
        isInteger: true,
      },
      {
        key: "visitorTimeOutTime",
        label: msg`Visitor Timeout`,
        description: msg`Timeout duration for visitor connections.`,
        min: 1,
        max: 600,
        step: 1,
        unit: "s",
        isInteger: false,
      },
      {
        key: "visitorDownloadSpan",
        label: msg`Visitor Download Span`,
        description: msg`Download span interval for visitor list updates.`,
        min: 1,
        max: 600,
        step: 1,
        unit: "s",
        isInteger: false,
      },
    ],
  },
];
