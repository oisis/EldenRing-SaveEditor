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
import type { SpectralSteedAttireStatus } from "../../application/world/worldPort";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Input } from "../../ui/components/Input/Input";
import { alert, message, panel, spacer, toolbar } from "../../ui/patterns/panel.css";
import {
  areaTitle,
  dataset as datasetBox,
  datasetBody,
  datasetSummary,
  datasetTitle,
  datasets,
  entry as entryBox,
  entryList,
  entryMeta,
  entryName,
  stepList,
  subnav,
} from "./WorldPanel.css";

/**
 * The World workspace, stage 9A: read-only.
 *
 * Every state, label, region and status on this screen is the backend's own
 * answer, carried as reported. Nothing here resolves an event flag, invents a
 * region for an entry the backend left unlabelled, derives a "current" quest
 * step out of several matched ones, or normalises a Spectral Steed `legacy` or
 * `conflict` status into one active attire.
 *
 * The screen offers no writer at all: no checkbox, no Unlock or Lock action, no
 * bulk operation and no disabled placeholder suggesting one is coming. A World
 * flag mutation needs the operation risk level, the risk reason and the
 * per-action capabilities the backend does not publish yet, and a pseudo-bulk
 * loop of single mutations could leave a partially changed save. Both belong to
 * stage 9B, together with the risk confirmation they require.
 *
 * Fog of War has a setter and no matching getter, so no control for it exists
 * here either: this screen never offers an action whose current state it cannot
 * show.
 */
export type WorldPanelProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
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

export function WorldPanel({ saveSessionID, saveRevision, characterID }: WorldPanelProps) {
  const { t } = useLingui();
  const [tab, setTab] = useState<WorldTab>("exploration");
  const [search, setSearch] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

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

  const displayName = (name: string) => (name === "" ? unnamedLabel : name);
  const joinFacts = (parts: readonly string[]) => parts.filter((part) => part !== "").join(" - ");

  const datasetsByTab: Record<WorldTab, PresentedDataset[]> = {
    exploration: [
      {
        id: "regions",
        title: t`Regions`,
        query: regions,
        active: regions.data?.active,
        entries: (regions.data?.regions ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.area,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
        })),
      },
      {
        id: "map-regions",
        title: t`Map Regions`,
        query: mapRegions,
        active: mapRegions.data?.active,
        entries: (mapRegions.data?.mapRegions ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.areaLabel,
          state: item.visible ? visibleLabel : hiddenLabel,
          done: item.visible,
          meta: "",
        })),
      },
      {
        id: "graces",
        title: t`Sites of Grace`,
        query: graces,
        active: graces.data?.active,
        entries: (graces.data?.graces ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.regionLabel,
          state: item.visited ? visitedLabel : notVisitedLabel,
          done: item.visited,
          meta: joinFacts([item.dungeonType, item.bossArena ? t`Boss arena` : ""]),
        })),
      },
      {
        id: "summoning-pools",
        title: t`Summoning Pools`,
        query: summoningPools,
        active: summoningPools.data?.active,
        entries: (summoningPools.data?.summoningPools ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.regionLabel,
          state: item.activated ? activatedLabel : notActivatedLabel,
          done: item.activated,
          meta: "",
        })),
      },
    ],
    progress: [
      {
        id: "bosses",
        title: t`Bosses`,
        query: bosses,
        active: bosses.data?.active,
        entries: (bosses.data?.bosses ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.regionLabel,
          state: item.defeated ? defeatedLabel : notDefeatedLabel,
          done: item.defeated,
          meta: joinFacts([item.encounterType, item.remembrance ? t`Remembrance` : ""]),
        })),
      },
      {
        id: "quests",
        title: t`Quests`,
        query: quests,
        active: quests.data?.active,
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
        entries: (gestures.data?.gestures ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.category,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
        })),
      },
      {
        id: "cookbooks",
        title: t`Cookbooks`,
        query: cookbooks,
        active: cookbooks.data?.active,
        entries: (cookbooks.data?.cookbooks ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.category,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
        })),
      },
      {
        id: "bell-bearings",
        title: t`Bell Bearings`,
        query: bellBearings,
        active: bellBearings.data?.active,
        entries: (bellBearings.data?.bellBearings ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: item.category,
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
        })),
      },
      {
        id: "whetblades",
        title: t`Whetblades`,
        query: whetblades,
        active: whetblades.data?.active,
        entries: (whetblades.data?.whetblades ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: "",
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
        })),
      },
      {
        id: "tutorials",
        title: t`Tutorials`,
        query: tutorials,
        active: tutorials.data?.active,
        entries: (tutorials.data?.tutorials ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.title),
          group: "",
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
        })),
      },
      {
        id: "colosseums",
        title: t`Colosseums`,
        query: colosseums,
        active: colosseums.data?.active,
        entries: (colosseums.data?.colosseums ?? []).map((item) => ({
          id: `${item.kind}/${item.key}`,
          name: displayName(item.name),
          group: "",
          state: item.unlocked ? unlockedLabel : lockedLabel,
          done: item.unlocked,
          meta: "",
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
        })),
      },
    ],
  };

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
