import { Trans, useLingui } from "@lingui/react/macro";
import { useRef, useState } from "react";
import { catalogAssetURL } from "../../application/catalog/catalogAssetURL";
import type {
  EquipmentCandidate,
  EquipmentMutationReceipt,
  LoadoutOwnedSlot,
  LoadoutResource,
  LoadoutSlot,
  LoadoutSpellSlot,
} from "../../application/equipment/equipmentPort";
import {
  useCharacterLoadout,
  useEquipmentCandidates,
} from "../../application/equipment/useEquipment";
import { useEquipmentMutations } from "../../application/equipment/useEquipmentMutations";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Input } from "../../ui/components/Input/Input";
import {
  alert,
  message,
  panel,
  spacer,
  toolbar,
  visuallyHidden,
} from "../../ui/patterns/panel.css";
import {
  board,
  candidateList,
  candidateName,
  candidate as candidateRow,
  group,
  rightGroup,
  leftGroup,
  ammoGroup,
  armorGroup,
  talismanGroup,
  quickGroup,
  pouchGroup,
  physickGroup,
  spellGroup,
  groupHeader,
  groupTitle,
  pagination,
  pickerSearch,
  pickerToolbar,
  pouchPad,
  slot as slotField,
  slotIcon,
  slotIconPlaceholder,
  slotMeta,
  slotName,
  slotRow,
  spellExtraGrid,
  spellPrimaryGrid,
} from "./EquipmentPanel.css";

/**
 * The Equipment workspace.
 *
 * `GetCharacterLoadout` is the only model this screen has: every name, icon,
 * slot state, locked position, capacity count and owned identity is the
 * backend's answer, carried as reported. Nothing here interprets a raw game ID,
 * recognises a sentinel, derives compatibility or recomputes Memory Slots
 * capacity — the picker shows exactly the page the backend served for the slot
 * type, and the setter it commits to validates the complete plan again.
 *
 * Ammunition is presented and never edited: no confirmed writer addresses the
 * arrow and bolt fields, so this screen offers no picker for them rather than
 * inventing one.
 */
export type EquipmentPanelProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  /**
   * The session controller's post-mutation step. It is absent while no session
   * controller is mounted, which is also when no mutation is offered.
   */
  applyMutationReceipt?: ((receipt: EquipmentMutationReceipt) => Promise<unknown>) | undefined;
  sessionBusy?: boolean;
};

const pickerPageSize = 24;

/** The groups a picker can be opened for, with the slot type each one asks for. */
type PickerTarget =
  | { group: "hands"; hand: "left" | "right"; index: number }
  | { group: "armor"; index: number }
  | { group: "talismans"; index: number }
  | { group: "quickItems"; index: number }
  | { group: "pouch"; index: number }
  | { group: "physick"; index: number }
  | { group: "spells"; index: number };

const armorSlotTypes = ["head", "chest", "arms", "legs"] as const;

function slotTypeOf(target: PickerTarget): string {
  switch (target.group) {
    case "hands":
      return target.hand === "left" ? "left_hand" : "right_hand";
    case "armor":
      return armorSlotTypes[target.index] ?? "";
    case "talismans":
      return "talisman";
    case "quickItems":
      return "quick_item";
    case "pouch":
      return "pouch";
    case "physick":
      return "physick";
    case "spells":
      return "spell_memory";
  }
}

/**
 * The owned identities of one positional group, in the backend's own order.
 *
 * `usable` is false when an occupied position carries no owned identity. That
 * cannot happen against the current backend contract, and if it ever did, the
 * group is shown read-only rather than committed with the position silently
 * cleared.
 */
function ownedAssignments(slots: readonly (LoadoutSlot | LoadoutOwnedSlot)[]): {
  values: (string | null)[];
  usable: boolean;
} {
  let usable = true;
  const values = slots.map((entry) => {
    if (entry.state !== "occupied") return null;
    if (entry.ownedItemID === undefined || entry.ownedItemID === "") {
      usable = false;
      return null;
    }
    return entry.ownedItemID;
  });
  return { values, usable };
}

/** The catalog references of one positional group, in the backend's own order. */
function resourceAssignments(
  slots: readonly (LoadoutSlot | LoadoutSpellSlot)[],
): (LoadoutResource | null)[] {
  return slots.map((entry) =>
    entry.state === "occupied" && entry.resource !== undefined
      ? { kind: entry.resource.kind, key: entry.resource.key }
      : null,
  );
}

function isNotNull<T>(value: T | null): value is T {
  return value !== null;
}

/**
 * The canonical identity of one catalog reference. Two resources are the same
 * one only when both halves of the reference match: a key is unique inside its
 * kind and not across kinds, so comparing keys alone would refuse an unrelated
 * resource and let a duplicate through.
 */
function resourceIdentity(resource: LoadoutResource): string {
  return `${resource.kind}/${resource.key}`;
}

/** The identities a resource-addressed group carries outside one position. */
function takenResourceIdentities(
  slots: readonly (LoadoutSlot | LoadoutSpellSlot)[],
  skip: number,
): ReadonlySet<string> {
  return new Set(
    resourceAssignments(slots)
      .map((entry, index) => (index === skip || entry === null ? null : resourceIdentity(entry)))
      .filter(isNotNull),
  );
}

export function EquipmentPanel({
  saveSessionID,
  saveRevision,
  characterID,
  applyMutationReceipt,
  sessionBusy = false,
}: EquipmentPanelProps) {
  const { t } = useLingui();
  const pickerReturnFocusRef = useRef<HTMLButtonElement | null>(null);
  // Without a session controller there is no receipt path, so the mutation
  // runner is given one that can never be reached: no picker is opened either.
  const mutations = useEquipmentMutations(applyMutationReceipt ?? (() => Promise.resolve()));

  const [target, setTarget] = useState<PickerTarget | null>(null);
  const [search, setSearch] = useState("");
  const [requestedPage, setRequestedPage] = useState(1);

  const loadout = useCharacterLoadout({ saveSessionID, saveRevision, characterID });
  const candidates = useEquipmentCandidates({
    saveSessionID,
    saveRevision,
    characterID,
    slotType: target === null ? undefined : slotTypeOf(target),
    search,
    page: requestedPage,
    pageSize: pickerPageSize,
  });

  const model = loadout.data;
  const canMutate =
    applyMutationReceipt !== undefined &&
    saveSessionID !== undefined &&
    saveRevision !== undefined &&
    characterID !== undefined &&
    model?.active === true &&
    !sessionBusy &&
    !mutations.isBusy;

  // The four armor positions, in the backend's own order.
  const armorLabels = [t`Head`, t`Chest`, t`Arms`, t`Legs`];
  // The confirmed Quick Pouch mapping of the 1.x view: the first four positions
  // are the four D-pad directions, the last two are ordinary fields. The label
  // is what a screen reader announces, so the direction is named rather than
  // left to the two-column shape alone.
  const pouchLabels = [
    t`Pouch up`,
    t`Pouch right`,
    t`Pouch left`,
    t`Pouch down`,
    t`Pouch 5`,
    t`Pouch 6`,
  ];
  const nameUnavailable = t`Name unavailable`;
  const emptyLabel = t`Empty`;
  const lockedLabel = t`Not unlocked yet`;

  /**
   * A group is editable only when every occupied position in it carries the
   * owned identity its setter needs. The backend always states one, so this is
   * a fail-closed guard: a group that lost it is shown read-only rather than
   * committed with the position silently cleared.
   */
  const groupEditable = (slots: readonly (LoadoutSlot | LoadoutOwnedSlot)[]) =>
    canMutate && ownedAssignments(slots).usable;

  const closePicker = () => {
    setTarget(null);
    setSearch("");
    setRequestedPage(1);
  };

  const openPicker = (next: PickerTarget, element: HTMLButtonElement) => {
    pickerReturnFocusRef.current = element;
    setSearch("");
    setRequestedPage(1);
    setTarget(next);
  };

  /**
   * Commits one picker choice. The complete group is assembled from the current
   * loadout and exactly one position is replaced, so a setter never receives a
   * partial group. The talisman and spell groups are compact by backend
   * contract, so their empty positions are dropped before the call.
   */
  const commit = async (chosen: EquipmentCandidate | null) => {
    if (target === null || model === undefined) return;
    if (saveSessionID === undefined || saveRevision === undefined || characterID === undefined) {
      return;
    }
    const scope = { saveSessionID, characterID, expectedRevision: saveRevision };
    const ownedItemID = chosen?.ownedItemID ?? null;
    const resource = chosen === null ? null : { ...chosen.resource };
    let applied = false;

    switch (target.group) {
      case "hands": {
        // The backend order is left 1, right 1, left 2, right 2, left 3, right 3.
        const left = ownedAssignments(model.leftHand);
        const right = ownedAssignments(model.rightHand);
        const next = left.values.flatMap((value, index) => [value, right.values[index] ?? null]);
        next[target.index * 2 + (target.hand === "left" ? 0 : 1)] = ownedItemID;
        applied = await mutations.setArmaments({ ...scope, slotAssignments: next });
        break;
      }
      case "armor": {
        const next = ownedAssignments(model.armor).values;
        next[target.index] = ownedItemID;
        applied = await mutations.setArmor({ ...scope, slotAssignments: next });
        break;
      }
      case "talismans": {
        const next = ownedAssignments(model.talismans).values;
        next[target.index] = ownedItemID;
        applied = await mutations.setTalismans({
          ...scope,
          orderedOwnedItemIDs: next.filter(isNotNull),
        });
        break;
      }
      case "quickItems": {
        const next = ownedAssignments(model.quickItems).values;
        next[target.index] = ownedItemID;
        applied = await mutations.setQuickItems({ ...scope, slotAssignments: next });
        break;
      }
      case "pouch": {
        const next = ownedAssignments(model.pouch).values;
        next[target.index] = ownedItemID;
        applied = await mutations.setPouch({ ...scope, slotAssignments: next });
        break;
      }
      case "physick": {
        // Both positions are sent as they are, so clearing one never left-packs
        // the other.
        const next = resourceAssignments(model.physick);
        next[target.index] = resource;
        applied = await mutations.setPhysick({ ...scope, crystalTearResources: next });
        break;
      }
      case "spells": {
        const next = resourceAssignments(model.spells);
        next[target.index] = resource;
        applied = await mutations.setSpells({
          ...scope,
          orderedResources: next.filter(isNotNull),
        });
        break;
      }
    }
    if (applied) closePicker();
  };

  /**
   * The identities the open group already carries at another position. The
   * setters reject the same record, talisman, spell or Crystal Tear twice, so
   * the picker refuses the choice before the call rather than after the
   * rejection. A record-addressed group is compared by owned identity and a
   * resource-addressed group by the complete canonical reference.
   */
  const takenElsewhere = (): ReadonlySet<string> => {
    if (target === null || model === undefined) return new Set();
    const collect = (values: readonly (string | null)[], skip: number) =>
      new Set(values.filter((value, index) => index !== skip && value !== null) as string[]);
    switch (target.group) {
      case "hands": {
        const values = [
          ...ownedAssignments(model.leftHand).values,
          ...ownedAssignments(model.rightHand).values,
        ];
        const skip = target.hand === "left" ? target.index : model.leftHand.length + target.index;
        return collect(values, skip);
      }
      case "armor":
        return collect(ownedAssignments(model.armor).values, target.index);
      case "talismans":
        return collect(ownedAssignments(model.talismans).values, target.index);
      case "quickItems":
        return collect(ownedAssignments(model.quickItems).values, target.index);
      case "pouch":
        return collect(ownedAssignments(model.pouch).values, target.index);
      case "spells":
        return takenResourceIdentities(model.spells, target.index);
      case "physick":
        // SetPhysickMixture rejects the same Crystal Tear in both positions, so
        // the picker refuses it before the call rather than after the rejection.
        return takenResourceIdentities(model.physick, target.index);
    }
  };

  /**
   * True when a spell obviously cannot fit the remaining capacity. The backend
   * stays the authority: this only spares the user a call that is certain to be
   * rejected, and a candidate whose cost the backend did not state is never
   * offered in the first place.
   */
  const exceedsMemory = (option: EquipmentCandidate): boolean => {
    if (target === null || target.group !== "spells" || model === undefined) return false;
    if (option.memorySlots === undefined) return false;
    const replaced = model.spells[target.index];
    const freed = replaced?.state === "occupied" ? (replaced.memorySlots ?? 0) : 0;
    return model.usedMemorySlots - freed + option.memorySlots > model.availableMemorySlots;
  };

  const handsEditable =
    model !== undefined && groupEditable(model.leftHand) && groupEditable(model.rightHand);

  // The identities the open group already carries, computed once per render
  // rather than once per candidate row.
  const takenIdentities = takenElsewhere();

  // The one rendering of a failed mutation. The interface owns the final
  // wording: the backend's own message is a developer fallback that may name a
  // host path or an internal identifier, so it is never shown. A failure leaves
  // the picker open so the user keeps their context, and the modal hides the
  // page behind it, so the same element is placed inside the dialog while one is
  // open.
  const mutationAlert = mutations.error ? (
    <p role="alert" className={alert}>
      <Trans>The change was not applied.</Trans>
    </p>
  ) : null;

  const displayName = (name: string | undefined) =>
    name === undefined || name === "" ? nameUnavailable : name;

  const renderSlot = (
    key: string,
    label: string,
    entry: { state: string; name?: string; iconPath?: string } | undefined,
    options: { onOpen?: (element: HTMLButtonElement) => void; note?: string } = {},
  ) => {
    const state = entry?.state ?? "empty";
    const locked = state === "locked";
    const occupied = state === "occupied";
    const content = locked ? lockedLabel : occupied ? displayName(entry?.name) : emptyLabel;
    const iconURL = occupied && entry?.iconPath ? catalogAssetURL(entry.iconPath) : undefined;
    const editable = options.onOpen !== undefined && !locked && canMutate;

    return (
      <Button
        key={key}
        className={slotField}
        disabled={!editable}
        aria-label={`${label}: ${content}`}
        onClick={(event) => options.onOpen?.(event.currentTarget)}
      >
        {iconURL !== undefined ? (
          <img className={slotIcon} src={iconURL} alt="" />
        ) : (
          <span className={slotIconPlaceholder} aria-hidden="true" />
        )}
        <span className={slotName}>{content}</span>
        <span className={slotMeta}>{options.note ?? label}</span>
      </Button>
    );
  };

  /** One spell position, wherever the grouping below places it. */
  const renderSpellSlot = (entry: LoadoutSpellSlot, index: number) =>
    renderSlot(`spell-${index}`, t`Spell ${index + 1}`, entry, {
      onOpen: (element) => openPicker({ group: "spells", index }, element),
      note:
        entry.state === "occupied" && entry.memorySlots !== undefined
          ? t`Spell ${index + 1} · ${entry.memorySlots} Memory Slots`
          : model?.activeSpellIndex === index
            ? t`Spell ${index + 1} · active`
            : undefined,
    });

  if (saveSessionID === undefined || characterID === undefined) {
    return (
      <Card aria-label={t`Equipment`} className={panel}>
        <p className={message}>
          <Trans>Open a save and select a character slot to edit its equipment.</Trans>
        </p>
      </Card>
    );
  }

  return (
    <Card aria-label={t`Equipment`} className={panel}>
      <div className={toolbar}>
        <span className={spacer} />
        {model?.active ? (
          <>
            <Badge>
              <Trans>
                Memory Slots: {model.usedMemorySlots} / {model.availableMemorySlots}
              </Trans>
            </Badge>
            <Badge>
              <Trans>Talisman slots: {model.unlockedTalismanSlots}</Trans>
            </Badge>
          </>
        ) : null}
      </div>

      {target === null ? mutationAlert : null}

      {loadout.isPending ? (
        <p role="status" className={message}>
          <Trans>Loading equipment…</Trans>
        </p>
      ) : null}
      {loadout.isError ? (
        <p role="alert" className={alert}>
          <Trans>Unable to load the equipment of this character slot.</Trans>
        </p>
      ) : null}
      {loadout.isSuccess && !loadout.data.active ? (
        <p className={message}>
          <Trans>This character slot is empty, so it has no equipment to show.</Trans>
        </p>
      ) : null}

      {model?.active ? (
        <div className={board}>
          <section className={`${group} ${rightGroup}`} aria-label={t`Right hand`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Right hand</Trans>
              </h3>
            </div>
            <div className={slotRow}>
              {model.rightHand.map((entry, index) =>
                renderSlot(`right-${index}`, t`Right hand ${index + 1}`, entry, {
                  onOpen: handsEditable
                    ? (element) => openPicker({ group: "hands", hand: "right", index }, element)
                    : undefined,
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${leftGroup}`} aria-label={t`Left hand`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Left hand</Trans>
              </h3>
            </div>
            <div className={slotRow}>
              {model.leftHand.map((entry, index) =>
                renderSlot(`left-${index}`, t`Left hand ${index + 1}`, entry, {
                  onOpen: handsEditable
                    ? (element) => openPicker({ group: "hands", hand: "left", index }, element)
                    : undefined,
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${ammoGroup}`} aria-label={t`Ammunition`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Ammunition</Trans>
              </h3>
              <Badge>
                <Trans>Read-only</Trans>
              </Badge>
            </div>
            <p className={message}>
              <Trans>Arrows and bolts are shown for reference and cannot be changed here.</Trans>
            </p>
            <div className={slotRow}>
              {model.arrows.map((entry, index) =>
                renderSlot(`arrow-${index}`, t`Arrows ${index + 1}`, entry),
              )}
              {model.bolts.map((entry, index) =>
                renderSlot(`bolt-${index}`, t`Bolts ${index + 1}`, entry),
              )}
            </div>
          </section>

          <section className={`${group} ${armorGroup}`} aria-label={t`Armor`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Armor</Trans>
              </h3>
            </div>
            <div className={slotRow}>
              {model.armor.map((entry, index) =>
                renderSlot(`armor-${index}`, armorLabels[index] ?? "", entry, {
                  onOpen: groupEditable(model.armor)
                    ? (element) => openPicker({ group: "armor", index }, element)
                    : undefined,
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${talismanGroup}`} aria-label={t`Talismans`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Talismans</Trans>
              </h3>
            </div>
            <p className={message}>
              <Trans>
                A locked position is one this character has not unlocked yet; the backend reports
                how many are available.
              </Trans>
            </p>
            <div className={slotRow}>
              {model.talismans.map((entry, index) =>
                renderSlot(`talisman-${index}`, t`Talisman ${index + 1}`, entry, {
                  onOpen: groupEditable(model.talismans)
                    ? (element) => openPicker({ group: "talismans", index }, element)
                    : undefined,
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${quickGroup}`} aria-label={t`Quick Items`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Quick Items</Trans>
              </h3>
            </div>
            <div className={slotRow}>
              {model.quickItems.map((entry, index) =>
                renderSlot(`quick-${index}`, t`Quick Item ${index + 1}`, entry, {
                  onOpen: groupEditable(model.quickItems)
                    ? (element) => openPicker({ group: "quickItems", index }, element)
                    : undefined,
                  note:
                    model.activeQuickItem === index
                      ? t`Quick Item ${index + 1} · active`
                      : undefined,
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${pouchGroup}`} aria-label={t`Quick Pouch`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Quick Pouch</Trans>
              </h3>
            </div>
            <div className={pouchPad}>
              {model.pouch.map((entry, index) =>
                renderSlot(`pouch-${index}`, pouchLabels[index] ?? "", entry, {
                  onOpen: groupEditable(model.pouch)
                    ? (element) => openPicker({ group: "pouch", index }, element)
                    : undefined,
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${physickGroup}`} aria-label={t`Wondrous Physick`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Wondrous Physick</Trans>
              </h3>
            </div>
            <div className={slotRow}>
              {model.physick.map((entry, index) =>
                renderSlot(`physick-${index}`, t`Crystal Tear ${index + 1}`, entry, {
                  onOpen: (element) => openPicker({ group: "physick", index }, element),
                }),
              )}
            </div>
          </section>

          <section className={`${group} ${spellGroup}`} aria-label={t`Spells`}>
            <div className={groupHeader}>
              <h3 className={groupTitle}>
                <Trans>Spells</Trans>
              </h3>
              <Badge>
                <Trans>
                  Memory Slots used: {model.usedMemorySlots} of {model.availableMemorySlots}
                </Trans>
              </Badge>
            </div>
            {/*
              The confirmed 1.x grouping: the first ten positions are two
              columns of five filled column by column, and the last two sit in
              their own two-column row below. The positions stay numbered — a
              spell slot has no D-pad direction.
            */}
            <div className={spellPrimaryGrid}>
              {model.spells.slice(0, 10).map((entry, index) => renderSpellSlot(entry, index))}
            </div>
            <div className={spellExtraGrid}>
              {model.spells.slice(10).map((entry, offset) => renderSpellSlot(entry, offset + 10))}
            </div>
          </section>
        </div>
      ) : null}

      <Dialog
        open={target !== null}
        onOpenChange={(open) => {
          if (!open) closePicker();
        }}
        title={<Trans>Choose an item</Trans>}
        description={
          <Trans>Only the items the backend accepts for this slot are offered here.</Trans>
        }
        closeLabel={<Trans>Cancel</Trans>}
        returnFocusRef={pickerReturnFocusRef}
      >
        {mutationAlert}
        <div className={pickerToolbar}>
          <label className={visuallyHidden} htmlFor="equipment-picker-search">
            <Trans>Search candidates</Trans>
          </label>
          <Input
            id="equipment-picker-search"
            className={pickerSearch}
            type="search"
            value={search}
            placeholder={t`Search candidates`}
            onChange={(event) => {
              const value = event.currentTarget.value;
              setSearch(value);
              setRequestedPage(1);
            }}
          />
          <Button size="sm" disabled={!canMutate} onClick={() => void commit(null)}>
            <Trans>Clear this slot</Trans>
          </Button>
        </div>

        {candidates.isPending ? (
          <p role="status" className={message}>
            <Trans>Loading candidates…</Trans>
          </p>
        ) : null}
        {candidates.isError ? (
          <p role="alert" className={alert}>
            <Trans>Unable to load the candidates for this slot.</Trans>
          </p>
        ) : null}
        {candidates.isSuccess && candidates.data.candidates.length === 0 ? (
          <p className={message}>
            <Trans>No item of this character can go into this slot.</Trans>
          </p>
        ) : null}

        {candidates.isSuccess && candidates.data.candidates.length > 0 ? (
          <ul className={candidateList}>
            {candidates.data.candidates.map((option) => {
              const taken =
                option.ownedItemID !== undefined
                  ? takenIdentities.has(option.ownedItemID)
                  : takenIdentities.has(resourceIdentity(option.resource));
              const refused = taken || exceedsMemory(option);
              const iconURL = catalogAssetURL(option.iconPath);
              return (
                <li key={`${option.resource.key}/${option.ownedItemID ?? ""}`}>
                  <Button
                    className={candidateRow}
                    disabled={!canMutate || refused}
                    onClick={() => void commit(option)}
                  >
                    {iconURL !== undefined ? (
                      <img className={slotIcon} src={iconURL} alt="" />
                    ) : (
                      <span className={slotIconPlaceholder} aria-hidden="true" />
                    )}
                    <span className={candidateName}>{displayName(option.name)}</span>
                    {option.quantity !== undefined ? (
                      <Badge>
                        <Trans>Quantity: {option.quantity}</Trans>
                      </Badge>
                    ) : null}
                    {option.memorySlots !== undefined ? (
                      <Badge>
                        <Trans>Memory Slots: {option.memorySlots}</Trans>
                      </Badge>
                    ) : null}
                    {option.banRisk ? (
                      <Badge tone="accent">
                        <Trans>Ban risk</Trans>
                      </Badge>
                    ) : null}
                    {taken ? (
                      <Badge>
                        <Trans>Already equipped</Trans>
                      </Badge>
                    ) : null}
                  </Button>
                </li>
              );
            })}
          </ul>
        ) : null}

        {candidates.isSuccess && candidates.data.total > candidates.data.pageSize ? (
          <nav className={pagination} aria-label={t`Candidate pages`}>
            <Button
              size="sm"
              disabled={candidates.data.page <= 1}
              onClick={() => setRequestedPage(candidates.data.page - 1)}
            >
              <Trans>Previous</Trans>
            </Button>
            <Badge>
              <Trans>Page {candidates.data.page}</Trans>
            </Badge>
            <Button
              size="sm"
              disabled={candidates.data.page * candidates.data.pageSize >= candidates.data.total}
              onClick={() => setRequestedPage(candidates.data.page + 1)}
            >
              <Trans>Next</Trans>
            </Button>
          </nav>
        ) : null}
      </Dialog>
    </Card>
  );
}
