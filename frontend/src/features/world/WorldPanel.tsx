import { Trans, useLingui } from "@lingui/react/macro";
import { type ReactNode, useMemo, useState } from "react";
import {
  useWorldBellBearings,
  useWorldBosses,
  useWorldColosseums,
  useWorldCookbooks,
  useWorldGestures,
  useWorldGraces,
  useWorldMapRegions,
  useWorldQuests,
  useWorldRegions,
  useWorldSpectralSteedAttires,
  useWorldSummoningPools,
  useWorldTutorials,
  useWorldWhetblades,
  type WorldQuery,
} from "../../application/world/useWorldViews";
import {
  findWorldCapability,
  useWorldMutationCapabilities,
  useWorldMutations,
  type WorldResourceToggleOperationKind,
} from "../../application/world/useWorldMutations";
import type {
  SpectralSteedAttireStatus,
  WorldMutationCapability,
  WorldMutationReceipt,
  WorldOperationKind,
} from "../../application/world/worldPort";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { alert, message, panel, spacer, toolbar } from "../../ui/patterns/panel.css";
import {
  areaTitle,
  dataset as datasetBox,
  datasetBody,
  datasetControls,
  datasetSummary,
  datasetTitle,
  datasets,
  entry as entryBox,
  entryList,
  entryMeta,
  entryName,
  stepList,
  stepPicker,
  subnav,
} from "./WorldPanel.css";

/**
 * The World workspace, stage 9B.
 *
 * Every state, label, region and status on this screen is the backend's own
 * answer, carried as reported. Nothing here resolves an event flag, invents a
 * region for an entry the backend left unlabelled, derives a "current" quest
 * step out of several matched ones, or normalises a Spectral Steed `legacy` or
 * `conflict` status into one active attire.
 *
 * The write side follows the same rule. An action exists only where the backend
 * returned the matching capability, and the risk level and risk reason shown
 * next to it are the backend's own values: this screen carries no risk table,
 * no availability rule and no bulk emulation of its own. Every action sends the
 * revision it was rendered for, and a failure leaves the view exactly as it
 * was, because nothing is applied optimistically.
 *
 * Three operations are not simple toggles and are not presented as ones:
 *
 *   - a quest step is chosen explicitly and applied, because `matched` is an
 *     independent per-step fact and never a current step to invert;
 *   - Fog of War is one-way. The backend accepts removal only and there is no
 *     getter for it, so the screen offers a single Remove action and no
 *     checkbox that would suggest a state it cannot show;
 *   - locking every Spectral Steed Attire is one atomic backend call, never a
 *     loop of single mutations.
 */
export type WorldPanelProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  /**
   * The session controller's own post-mutation step. Without it the workspace
   * stays read-only: a committed change that nothing published would leave the
   * rest of the application on a stale revision.
   */
  applyMutationReceipt?: ((receipt: WorldMutationReceipt) => Promise<unknown>) | undefined;
  /** True while the session itself is busy; every World action waits for it. */
  sessionBusy?: boolean | undefined;
};

type WorldTab = "exploration" | "progress" | "unlocks";

/** One presented entry of one dataset, already reduced to what the list shows. */
type PresentedEntry = {
  id: string;
  name: string;
  /** The backend-supplied grouping label, or the empty string when it gave none. */
  group: string;
  /** The state wording shown for the entry. */
  state: string;
  /**
   * Whether the entry counts as completed. It is `undefined` for a dataset
   * without a single unambiguous boolean, which is what keeps a completion
   * counter from being invented for one.
   */
  done: boolean | undefined;
  /** Extra backend facts shown under the name, already localised. */
  meta: string;
  details?: ReactNode;
  /**
   * The writer of this entry, already built from the backend capability. It is
   * absent when the backend published no capability for the operation or when
   * nothing may be written right now.
   */
  action?: ReactNode;
};

/** The shape the screen renders one dataset from, whatever getter produced it. */
type PresentedDataset = {
  id: string;
  title: string;
  /** Only the parts of a query result this screen reacts to. */
  query: { isPending: boolean; isError: boolean; isSuccess: boolean };
  /** The backend's own slot state, unknown until the answer arrives. */
  active: boolean | undefined;
  entries: readonly PresentedEntry[];
  /** A dataset-level badge, such as the Spectral Steed status. */
  note?: string;
  /** The backend risk line of this dataset's operation, and any bulk action. */
  controls?: ReactNode;
};

/** The neutral bucket for an entry the backend supplied no label for. */
const otherGroup = " other";


function groupEntries(entries: readonly PresentedEntry[]): [string, PresentedEntry[]][] {
  const groups = new Map<string, PresentedEntry[]>();
  for (const item of entries) {
    const key = item.group === "" ? otherGroup : item.group;
    const bucket = groups.get(key);
    if (bucket === undefined) {
      groups.set(key, [item]);
    } else {
      bucket.push(item);
    }
  }
  return [...groups];
}

export function WorldPanel({
  saveSessionID,
  saveRevision,
  characterID,
  applyMutationReceipt,
  sessionBusy,
}: WorldPanelProps) {
  const { t } = useLingui();
  const [tab, setTab] = useState<WorldTab>("exploration");
  const [search, setSearch] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  // The edit context key binds uncommitted input to the exact session, character
  // and revision it was chosen for, so a choice belonging to another context is
  // never rendered for or applied to a different character or revision.
  const editContextKey = `${saveSessionID ?? ""}|${characterID ?? ""}|${saveRevision ?? ""}`;

  // The step one questline is about to receive, by questline. It is a choice
  // the user made and never a step derived from `matched`.
  const [selectedStepDraft, setSelectedStepDraft] = useState<
    { key: string; value: Record<string, string> } | undefined
  >(undefined);
  const selectedStep =
    selectedStepDraft?.key === editContextKey ? selectedStepDraft.value : undefined;

  const query: WorldQuery = { saveSessionID, saveRevision, characterID };

  const regions = useWorldRegions(query);
  const mapRegions = useWorldMapRegions(query);
  const graces = useWorldGraces(query);
  const summoningPools = useWorldSummoningPools(query);
  const bosses = useWorldBosses(query);
  const quests = useWorldQuests(query);
  const gestures = useWorldGestures(query);
  const cookbooks = useWorldCookbooks(query);
  const bellBearings = useWorldBellBearings(query);
  const whetblades = useWorldWhetblades(query);
  const tutorials = useWorldTutorials(query);
  const colosseums = useWorldColosseums(query);
  const attires = useWorldSpectralSteedAttires(query);

  const capabilities = useWorldMutationCapabilities();
  const mutations = useWorldMutations(applyMutationReceipt ?? (() => Promise.resolve()));

  // The revision an action is rendered for travels with it. Without a complete
  // scope there is nothing to write against, so no writer is rendered at all;
  // a missing value is never read as a default.
  const scope =
    saveSessionID === undefined || saveRevision === undefined || characterID === undefined
      ? undefined
      : { saveSessionID, characterID, expectedRevision: saveRevision };

  // Writing is possible only while the session accepts one: the controller's
  // post-mutation step exists, the session is idle, no World mutation is in
  // flight and the backend contract itself has arrived.
  const writable =
    scope !== undefined &&
    applyMutationReceipt !== undefined &&
    sessionBusy !== true &&
    !mutations.isBusy &&
    capabilities.isSuccess;

  const capability = (operationKind: WorldOperationKind): WorldMutationCapability | undefined =>
    findWorldCapability(capabilities.data, operationKind);

  /**
   * The risk line of one operation, and the reason the backend gave for it.
   * Both are shown before the operation runs and neither is worded here: the
   * reason is the backend's own sentence, displayed verbatim.
   */
  const riskLine = (found: WorldMutationCapability | undefined): ReactNode =>
    found === undefined ? null : (
      <p className={entryMeta}>{t`Risk: ${found.risk} - ${found.riskReason}`}</p>
    );

  /**
   * The writer of one resource toggle. It exists only where the backend
   * published the capability, and it sends the exact opposite of the state the
   * backend reported for that entry, under the revision it was rendered for.
   */
  const toggleAction = (
    operationKind: WorldResourceToggleOperationKind,
    item: { kind: string; key: string; name: string },
    current: boolean,
    targetState: string,
    ready: boolean,
  ): ReactNode => {
    if (scope === undefined || capability(operationKind) === undefined) return null;
    return (
      <Button
        size="sm"
        disabled={!writable || !ready}
        aria-label={`${displayName(item.name)}: ${targetState}`}
        onClick={() =>
          void mutations.toggleResource(operationKind, {
            ...scope,
            resourceKind: item.kind,
            resourceKey: item.key,
            value: !current,
          })
        }
      >
        {targetState}
      </Button>
    );
  };

  const unlockedLabel = t`Unlocked`;
  const lockedLabel = t`Locked`;
  const visibleLabel = t`Revealed`;
  const hiddenLabel = t`Not revealed`;
  const visitedLabel = t`Visited`;
  const notVisitedLabel = t`Not visited`;
  const defeatedLabel = t`Defeated`;
  const notDefeatedLabel = t`Not defeated`;
  const activatedLabel = t`Activated`;
  const notActivatedLabel = t`Not activated`;
  const ownedLabel = t`Owned`;
  const notOwnedLabel = t`Not owned`;
  const unnamedLabel = t`Name unavailable`;
  const otherLabel = t`Other`;

  // The wording of the state an action writes, which is the exact opposite of
  // the state the backend reported for that entry.
  const setUnlockedLabel = t`Set unlocked`;
  const setLockedLabel = t`Set locked`;
  const setRevealedLabel = t`Set revealed`;
  const setHiddenLabel = t`Set not revealed`;
  const setVisitedLabel = t`Set visited`;
  const setNotVisitedLabel = t`Set not visited`;
  const setDefeatedLabel = t`Set defeated`;
  const setNotDefeatedLabel = t`Set not defeated`;
  const setActivatedLabel = t`Set activated`;
  const setNotActivatedLabel = t`Set not activated`;

  const questsReady = quests.isSuccess && quests.data?.active === true;
  const attiresReady = attires.isSuccess && attires.data?.active === true;

  const displayName = (name: string) => (name === "" ? unnamedLabel : name);
  const joinFacts = (parts: readonly string[]) => parts.filter((part) => part !== "").join(" - ");

  const datasetsByTab: Record<WorldTab, PresentedDataset[]> = {
    exploration: [
      {
        id: "regions",
        title: t`Regions`,
        query: regions,
        active: regions.data?.active,
        controls: riskLine(capability("set_region_unlocked")),
        entries: (regions.data?.regions ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.area,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_region_unlocked",
            item,
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            regions.isSuccess && regions.data?.active === true,
          ),
        })),
      },
      {
        id: "map-regions",
        title: t`Map Regions`,
        query: mapRegions,
        active: mapRegions.data?.active,
        controls: riskLine(capability("set_map_region_revealed")),
        entries: (mapRegions.data?.mapRegions ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.areaLabel,
          state: item.visible ? visibleLabel : hiddenLabel,
          done: item.visible,
          meta: "",
          action: toggleAction(
            "set_map_region_revealed",
            item,
            item.visible,
            item.visible ? setHiddenLabel : setRevealedLabel,
            mapRegions.isSuccess && mapRegions.data?.active === true,
          ),
        })),
      },
      {
        id: "graces",
        title: t`Sites of Grace`,
        query: graces,
        active: graces.data?.active,
        controls: riskLine(capability("set_grace_visited")),
        entries: (graces.data?.graces ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.regionLabel,
          state: item.visited ? visitedLabel : notVisitedLabel,
          done: item.visited,
          meta: joinFacts([item.dungeonType, item.bossArena ? t`Boss arena` : ""]),
          action: toggleAction(
            "set_grace_visited",
            item,
            item.visited,
            item.visited ? setNotVisitedLabel : setVisitedLabel,
            graces.isSuccess && graces.data?.active === true,
          ),
        })),
      },
      {
        id: "summoning-pools",
        title: t`Summoning Pools`,
        query: summoningPools,
        active: summoningPools.data?.active,
        controls: riskLine(capability("set_summoning_pool_activated")),
        entries: (summoningPools.data?.summoningPools ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.regionLabel,
          state: item.activated ? activatedLabel : notActivatedLabel,
          done: item.activated,
          meta: "",
          action: toggleAction(
            "set_summoning_pool_activated",
            item,
            item.activated,
            item.activated ? setNotActivatedLabel : setActivatedLabel,
            summoningPools.isSuccess && summoningPools.data?.active === true,
          ),
        })),
      },
    ],
    progress: [
      {
        id: "bosses",
        title: t`Bosses`,
        query: bosses,
        active: bosses.data?.active,
        controls: riskLine(capability("set_boss_defeated")),
        entries: (bosses.data?.bosses ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.regionLabel,
          state: item.defeated ? defeatedLabel : notDefeatedLabel,
          done: item.defeated,
          meta: joinFacts([item.encounterType, item.remembrance ? t`Remembrance` : ""]),
          action: toggleAction(
            "set_boss_defeated",
            item,
            item.defeated,
            item.defeated ? setNotDefeatedLabel : setDefeatedLabel,
            bosses.isSuccess && bosses.data?.active === true,
          ),
        })),
      },
      {
        id: "quests",
        title: t`Quests`,
        query: quests,
        active: quests.data?.active,
        controls: riskLine(capability("set_quest_step")),
        // A questline has no completion boolean: the backend reports one
        // independent `matched` fact per step and can match several at once, so
        // the entry states matched steps out of total steps and no step is
        // promoted to a "current" one.
        entries: (quests.data?.quests ?? []).map((item) => {
          const matched = item.steps.filter((step) => step.matched).length;
          return {
            id: `${item.kind}/${item.key}`,
            name: displayName(item.name),
            group: "",
            state: t`${matched} of ${item.steps.length} matched steps`,
            done: undefined,
            meta: "",
            details: (
              <ul className={stepList}>
                {item.steps.map((step) => (
                  <li key={`${step.stepKind}/${step.stepKey}`} className={entryMeta}>
                    {joinFacts([
                      step.matched ? t`Matched` : t`Not matched`,
                      displayName(step.description),
                      step.location,
                    ])}
                  </li>
                ))}
              </ul>
            ),
            // A questline is not a toggle: the step is picked explicitly and
            // applied. `matched` says which steps the save already satisfies
            // and is never turned into a current step that could be inverted,
            // so the picker starts empty and Apply stays disabled until a step
            // is chosen.
            action:
              scope === undefined || capability("set_quest_step") === undefined ? null : (
                <div className={stepPicker}>
                  <Select
                    aria-label={t`${displayName(item.name)}: quest step`}
                    value={selectedStep?.[`${item.kind}/${item.key}`] ?? ""}
                    disabled={!writable || !questsReady}
                    onChange={(event) => {
                      const chosen = event.currentTarget.value;
                      setSelectedStepDraft({
                        key: editContextKey,
                        value: {
                          ...(selectedStep ?? {}),
                          [`${item.kind}/${item.key}`]: chosen,
                        },
                      });
                    }}
                  >
                    <option value="">{t`Select a step`}</option>
                    {item.steps.map((step) => (
                      <option
                        key={`${step.stepKind}/${step.stepKey}`}
                        value={`${step.stepKind}/${step.stepKey}`}
                      >
                        {displayName(step.description)}
                      </option>
                    ))}
                  </Select>
                  <Button
                    size="sm"
                    aria-label={t`${displayName(item.name)}: apply step`}
                    disabled={
                      !writable ||
                      !questsReady ||
                      (selectedStep?.[`${item.kind}/${item.key}`] ?? "") === ""
                    }
                    onClick={() => {
                      const chosen = item.steps.find(
                        (step) =>
                          `${step.stepKind}/${step.stepKey}` ===
                          selectedStep?.[`${item.kind}/${item.key}`],
                      );
                      if (chosen === undefined) return;
                      void mutations.applyQuestStep({
                        ...scope,
                        questKind: item.kind,
                        questKey: item.key,
                        stepKind: chosen.stepKind,
                        stepKey: chosen.stepKey,
                      });
                    }}
                  >
                    <Trans>Apply</Trans>
                  </Button>
                </div>
              ),
          };
        }),
      },
    ],
    unlocks: [
      {
        id: "gestures",
        title: t`Gestures`,
        query: gestures,
        active: gestures.data?.active,
        controls: riskLine(capability("set_gesture_unlocked")),
        entries: (gestures.data?.gestures ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.category,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_gesture_unlocked",
            item,
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            gestures.isSuccess && gestures.data?.active === true,
          ),
        })),
      },
      {
        id: "cookbooks",
        title: t`Cookbooks`,
        query: cookbooks,
        active: cookbooks.data?.active,
        controls: riskLine(capability("set_cookbook_unlocked")),
        entries: (cookbooks.data?.cookbooks ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.category,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_cookbook_unlocked",
            item,
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            cookbooks.isSuccess && cookbooks.data?.active === true,
          ),
        })),
      },
      {
        id: "bell-bearings",
        title: t`Bell Bearings`,
        query: bellBearings,
        active: bellBearings.data?.active,
        controls: riskLine(capability("set_bell_bearing_unlocked")),
        entries: (bellBearings.data?.bellBearings ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.category,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_bell_bearing_unlocked",
            item,
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            bellBearings.isSuccess && bellBearings.data?.active === true,
          ),
        })),
      },
      {
        id: "whetblades",
        title: t`Whetblades`,
        query: whetblades,
        active: whetblades.data?.active,
        controls: riskLine(capability("set_whetblade_unlocked")),
        entries: (whetblades.data?.whetblades ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: "",
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_whetblade_unlocked",
            item,
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            whetblades.isSuccess && whetblades.data?.active === true,
          ),
        })),
      },
      {
        id: "tutorials",
        title: t`Tutorials`,
        query: tutorials,
        active: tutorials.data?.active,
        controls: riskLine(capability("set_tutorial_unlocked")),
        entries: (tutorials.data?.tutorials ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.title),
          group: "",
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_tutorial_unlocked",
            // The tutorial entry names its label `title`; the pair the endpoint
            // addresses is the same kind and key every other entry carries.
            { ...item, name: item.title },
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            tutorials.isSuccess && tutorials.data?.active === true,
          ),
        })),
      },
      {
        id: "colosseums",
        title: t`Colosseums`,
        query: colosseums,
        active: colosseums.data?.active,
        controls: riskLine(capability("set_colosseum_unlocked")),
        entries: (colosseums.data?.colosseums ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: "",
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
          action: toggleAction(
            "set_colosseum_unlocked",
            item,
            item.unlocked,
            item.unlocked ? setLockedLabel : setUnlockedLabel,
            colosseums.isSuccess && colosseums.data?.active === true,
          ),
        })),
      },
      {
        id: "spectral-steed-attires",
        title: t`Spectral Steed Attires`,
        query: attires,
        active: attires.data?.active,
        // The backend's own classification. `legacy` and `conflict` are answers,
        // not errors, and neither is reduced here to one active attire; the
        // counter describes ownership, which is what the entries state.
        // The bulk lock is one atomic backend call and is offered only where
        // the backend declared it supports one. Every other World change is a
        // single-target write, so nothing here loops setters to fake a set
        // operation.
        controls: (
          <>
            {riskLine(capability("set_spectral_steed_attire"))}
            {scope !== undefined && capability("lock_all_spectral_steed_attires")?.supportsBulk ? (
              <div className={datasetControls}>
                {riskLine(capability("lock_all_spectral_steed_attires"))}
                <Button
                  size="sm"
                  disabled={!writable || !attiresReady}
                  onClick={() => void mutations.lockAllSpectralSteedAttires(scope)}
                >
                  <Trans>Lock all Spectral Steed Attires</Trans>
                </Button>
              </div>
            ) : null}
          </>
        ),
        note: spectralSteedStatusLabel(attires.data?.status, {
          resolved: t`Status: resolved`,
          legacy: t`Status: legacy`,
          conflict: t`Status: conflict`,
        }),
        entries: (attires.data?.attires ?? []).map((item) => ({
          id: item.attireKey,
          name: displayName(item.name),
          group: "",
          state: item.owned ? ownedLabel : notOwnedLabel,
          done: item.owned,
          meta:
            attires.data?.status === "resolved" &&
            attires.data.activeAttireKey === item.attireKey &&
            item.attireKey !== ""
              ? t`Active`
              : "",
          // An appearance can be selected only where the backend reported it
          // owned. The backend is the contract owner and reports the default
          // appearance as owned=true; nothing unowned is presented as a choice.
          action:
            scope === undefined ||
            capability("set_spectral_steed_attire") === undefined ||
            !item.owned ? null : (
              <Button
                size="sm"
                aria-label={t`${displayName(item.name)}: select appearance`}
                disabled={!writable || !attiresReady}
                onClick={() =>
                  void mutations.selectSpectralSteedAttire({
                    ...scope,
                    attireKey: item.attireKey,
                  })
                }
              >
                <Trans>Select</Trans>
              </Button>
            ),
        })),
      },
    ],
  };

  /**
   * Fog of War is one-way by contract: the backend accepts removal only and
   * publishes no getter for it, so the screen offers a single Remove action,
   * shows no checkbox and no pretended current state, and never sends `false`.
   */
  const fogOfWarAction =
    scope === undefined || capability("set_fog_of_war_removed") === undefined ? null : (
      <section aria-label={t`Fog of War`} className={datasetControls}>
        <h3 className={datasetTitle}>
          <Trans>Fog of War</Trans>
        </h3>
        {riskLine(capability("set_fog_of_war_removed"))}
        <Button size="sm" disabled={!writable} onClick={() => void mutations.removeFogOfWar(scope)}>
          <Trans>Remove Fog of War</Trans>
        </Button>
      </section>
    );

  const visible = datasetsByTab[tab];

  const needle = search.trim().toLocaleLowerCase();
  // The search narrows the rendered list only. The dataset itself stays whole,
  // because the counter describes the backend's complete answer for the section
  // and must not change while the user is looking for one entry.
  const searched = useMemo(
    () =>
      visible.map((set) => ({
        set,
        shown:
          needle === ""
            ? set.entries
            : set.entries.filter((item) =>
                `${item.name} ${item.group} ${item.meta}`.toLocaleLowerCase().includes(needle),
              ),
      })),
    [visible, needle],
  );

  const setAllCollapsed = (value: boolean) =>
    setCollapsed((current) => {
      const next = { ...current };
      for (const set of visible) {
        next[set.id] = value;
      }
      return next;
    });

  if (saveSessionID === undefined || saveRevision === undefined || characterID === undefined) {
    return (
      <Card aria-label={t`World`} className={panel}>
        <p className={message}>
          <Trans>Open a save and select a character slot to see its World progress.</Trans>
        </p>
      </Card>
    );
  }

  return (
    <Card aria-label={t`World`} className={panel}>
      <nav aria-label={t`World sections`} className={subnav}>
        <Button size="sm" pressed={tab === "exploration"} onClick={() => setTab("exploration")}>
          <Trans>Exploration</Trans>
        </Button>
        <Button size="sm" pressed={tab === "progress"} onClick={() => setTab("progress")}>
          <Trans>Progress</Trans>
        </Button>
        <Button size="sm" pressed={tab === "unlocks"} onClick={() => setTab("unlocks")}>
          <Trans>Unlocks</Trans>
        </Button>
      </nav>

      <div className={toolbar}>
        <Input
          type="search"
          aria-label={t`Search World entries`}
          value={search}
          onChange={(event) => setSearch(event.currentTarget.value)}
        />
        <span className={spacer} />
        <Button size="sm" onClick={() => setAllCollapsed(false)}>
          <Trans>Expand all</Trans>
        </Button>
        <Button size="sm" onClick={() => setAllCollapsed(true)}>
          <Trans>Collapse all</Trans>
        </Button>
      </div>

      {mutations.error ? (
        <p role="alert" className={alert}>
          <Trans>The change was not applied.</Trans>
        </p>
      ) : null}

      {tab === "exploration" ? fogOfWarAction : null}

      <div className={datasets}>
        {searched.map(({ set, shown }) => {
          // Only a dataset with an unambiguous boolean per entry is counted, and
          // only once its own answer has arrived for an active slot: a counter
          // must never describe a pending, failed or empty slot.
          const counted = set.entries.filter((item) => item.done !== undefined);
          const completed = counted.filter((item) => item.done === true).length;
          const ready = set.query.isSuccess && set.active === true;

          return (
            <details
              key={set.id}
              aria-label={set.title}
              className={datasetBox}
              open={collapsed[set.id] !== true}
              onToggle={(event) => {
                // The open state is read while the event is still live: a
                // synthetic event may no longer carry its target by the time
                // the state updater runs.
                const isOpen = event.currentTarget.open;
                setCollapsed((current) => ({ ...current, [set.id]: !isOpen }));
              }}
            >
              <summary className={datasetSummary}>
                <h3 className={datasetTitle}>{set.title}</h3>
                {ready && counted.length > 0 ? (
                  <Badge>
                    {completed} / {counted.length}
                  </Badge>
                ) : null}
                {ready && set.note !== undefined ? <Badge>{set.note}</Badge> : null}
              </summary>
              <div className={datasetBody}>
                {ready && set.controls !== undefined ? set.controls : null}
                {set.query.isPending ? (
                  <p role="status" className={message}>
                    <Trans>Loading…</Trans>
                  </p>
                ) : null}
                {set.query.isError ? (
                  <p role="alert" className={alert}>
                    <Trans>Unable to load this World section.</Trans>
                  </p>
                ) : null}
                {set.query.isSuccess && set.active !== true ? (
                  <p className={message}>
                    <Trans>
                      This character slot is empty, so it has no World progress to show.
                    </Trans>
                  </p>
                ) : null}
                {ready && shown.length === 0 ? (
                  <p className={message}>
                    <Trans>No entry matches the search.</Trans>
                  </p>
                ) : null}
                {ready
                  ? groupEntries(shown).map(([group, items]) => {
                      const label = group === otherGroup ? otherLabel : group;
                      return (
                        <section key={group} aria-label={label}>
                          <h4 className={areaTitle}>{label}</h4>
                          <ul className={entryList}>
                            {items.map((item) => (
                              <li key={item.id} className={entryBox}>
                                <span className={entryName}>{item.name}</span>
                                <span className={entryMeta}>{item.state}</span>
                                {item.meta === "" ? null : (
                                  <span className={entryMeta}>{item.meta}</span>
                                )}
                                {item.details}
                                {item.action}
                              </li>
                            ))}
                          </ul>
                        </section>
                      );
                    })
                  : null}
              </div>
            </details>
          );
        })}
      </div>
    </Card>
  );
}

/**
 * The badge wording of the Spectral Steed status. The status is a closed
 * contract validated at the bridge boundary, so this screen only has to word
 * the three states it knows and never presents a raw backend value.
 */
function spectralSteedStatusLabel(
  status: SpectralSteedAttireStatus | undefined,
  labels: Record<SpectralSteedAttireStatus, string>,
): string | undefined {
  return status === undefined ? undefined : labels[status];
}
