import { Trans, useLingui } from "@lingui/react/macro";
import { type RefObject, useCallback, useEffect, useRef, useState } from "react";
import { useCatalogResources } from "../../application/catalog/useCatalogResources";
import type { CharacterSlot, CharacterSummary } from "../../application/character/characterPort";
import {
  type CharacterSlotMutations,
  type SlotMutationOutcome,
  useCharacterSlotMutations,
} from "../../application/character/useCharacterSlotMutations";
import type { MutationReceipt } from "../../application/save-session/saveSessionPort";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Select } from "../../ui/components/Select/Select";
import { alertPanel, message } from "../../ui/patterns/panel.css";
import {
  dialogActions,
  dialogSection,
  disclosure,
  entry,
  group,
  groupTitle,
  inactiveRow,
  level as levelText,
  list,
  manage,
  meta,
  name as nameText,
  row,
  rowHead,
  sidebar,
  stateLabel,
} from "./CharacterSidebar.css";
import type { CharacterSelection } from "./useCharacterSelection";

/**
 * The session the slot operations run against. Without it the panel stays a
 * read-only list: no management control is rendered at all, which is what the
 * Home screen and the component tests use.
 */
export type SlotManagement = {
  saveSessionID: string;
  /** The revision every operation is sent with; the backend rejects a stale one. */
  saveRevision: string;
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<unknown>;
  /** The session already runs another operation; slot operations wait for it. */
  sessionBusy?: boolean;
};

/**
 * One opening of the slot manager. The token separates two openings of the same
 * slot, so a request started in the first one can never act on the second.
 */
type ManageIntent = {
  characterID: number;
  token: number;
};

/**
 * The character panel: ten physical slots, active ones before inactive ones.
 *
 * Everything it presents comes from the backend. The name and rune level come
 * from the character summary, and the slot state, the starting class and the
 * offered operations come from the slot projection of the same read. The panel
 * classifies nothing itself: it never infers `Empty` from `inactive`, never
 * invents a starting class for a slot the backend could not read one for, and
 * never offers an operation the backend did not report as available.
 */
export function CharacterSidebar({
  model,
  management,
}: {
  model: CharacterSelection;
  management?: SlotManagement;
}) {
  const { t } = useLingui();
  const { hasSession, characters, activeCharacters, inactiveCharacters } = model;
  const [inactiveOpen, setInactiveOpen] = useState(false);
  const [manage, setManage] = useState<ManageIntent | undefined>(undefined);
  const menuRef = useRef<HTMLButtonElement | null>(null);
  const openCount = useRef(0);

  // The in-flight lock lives here and not inside the dialog: closing the dialog
  // must not release it, and reopening it must not create a second, free one.
  const applyMutationReceipt = management?.applyMutationReceipt;
  const applyReceipt = useCallback(
    (receipt: MutationReceipt) =>
      applyMutationReceipt === undefined
        ? Promise.reject(new Error("no_save_session"))
        : applyMutationReceipt(receipt),
    [applyMutationReceipt],
  );
  // What the slots on screen were read for. A committed mutation that could not
  // be reflected stays blocking until this states the revision it committed.
  const listing = model.characters.data;
  const mutations = useCharacterSlotMutations(
    applyReceipt,
    management === undefined
      ? undefined
      : {
          saveSessionID: management.saveSessionID,
          readRevision:
            listing?.saveSessionID === management.saveSessionID ? listing.saveRevision : undefined,
        },
  );

  const openManage = (characterID: number, button: HTMLButtonElement) => {
    menuRef.current = button;
    mutations.clearError();
    openCount.current += 1;
    setManage({ characterID, token: openCount.current });
  };

  // A finished request may only close the dialog it was started from. The same
  // slot opened again is a new intent and keeps its own dialog open.
  const closeManage = useCallback(
    (token: number) => setManage((current) => (current?.token === token ? undefined : current)),
    [],
  );

  const classes = useCatalogResources({
    resourceType: "class",
    family: "",
    capability: "",
    endpointID: "",
    search: "",
    page: 1,
    pageSize: 50,
  });

  /**
   * The starting class of one slot, or `undefined` when it cannot be named
   * safely. An unread catalog and a slot the backend reported no class for are
   * the same answer: no class is shown rather than a default one.
   */
  const classNameOf = (slot: CharacterSlot | undefined) => {
    if (slot === undefined || !slot.startingClassKnown || !classes.isSuccess) return undefined;
    return classes.data.resources.find((resource) => resource.key === String(slot.startingClassID))
      ?.name;
  };

  return (
    <aside aria-label={t`Characters`} className={sidebar}>
      {!hasSession && (
        <p className={message}>
          <Trans>No save loaded.</Trans>
        </p>
      )}

      {hasSession && characters.isPending && (
        <p role="status" className={message}>
          <Trans>Loading characters…</Trans>
        </p>
      )}

      {hasSession && characters.isError && (
        // The transport error never reaches the interface: the adapter reduces
        // every failure to one code, so the user sees one safe message.
        <p role="alert" className={alertPanel}>
          <Trans>Unable to load characters.</Trans>
        </p>
      )}

      {characters.isSuccess && activeCharacters.length === 0 && (
        <p className={message}>
          <Trans>No active character is available.</Trans>
        </p>
      )}

      {activeCharacters.length > 0 && (
        <section className={group}>
          <h3 className={groupTitle}>
            <Trans>Active characters</Trans>
          </h3>
          <ul className={list}>
            {activeCharacters.map((character) => (
              <li key={character.characterID} className={entry}>
                <ActiveRow
                  character={character}
                  startingClass={classNameOf(model.slotOf(character.characterID))}
                  selected={model.selectedCharacterID === character.characterID}
                  onSelect={() => model.selectCharacter(character.characterID)}
                />
                {management !== undefined && (
                  <ManageButton
                    label={t`Manage slot — ${character.name}`}
                    onOpen={(button) => openManage(character.characterID, button)}
                  />
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      {inactiveCharacters.length > 0 && (
        <section className={group}>
          <Button
            className={disclosure}
            aria-expanded={inactiveOpen}
            onClick={() => setInactiveOpen((open) => !open)}
          >
            <span aria-hidden="true">{inactiveOpen ? "▾" : "▸"}</span>
            <InactiveCount count={inactiveCharacters.length} />
          </Button>
          {inactiveOpen && (
            <ul className={list}>
              {inactiveCharacters.map((character) => {
                const slot = model.slotOf(character.characterID);
                return (
                  <li key={character.characterID} className={entry}>
                    {/* Not a control: an inactive slot carries no selection. It
                        states the backend's classification and nothing else. */}
                    <div className={inactiveRow}>
                      <span className={stateLabel}>
                        <SlotStateLabel slot={slot} />
                      </span>
                      <span className={meta}>
                        <SlotMeta
                          characterID={character.characterID}
                          startingClass={classNameOf(slot)}
                        />
                      </span>
                    </div>
                    {management !== undefined && (
                      <ManageButton
                        label={t`Manage slot ${character.characterID + 1}`}
                        onOpen={(button) => openManage(character.characterID, button)}
                      />
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      )}

      {management !== undefined && manage !== undefined && (
        <SlotManagerDialog
          key={`${management.saveSessionID}:${manage.characterID}:${manage.token}`}
          management={management}
          model={model}
          mutations={mutations}
          characterID={manage.characterID}
          token={manage.token}
          startingClass={classNameOf(model.slotOf(manage.characterID))}
          returnFocusRef={menuRef}
          onClose={closeManage}
        />
      )}
    </aside>
  );
}

function InactiveCount({ count }: { count: number }) {
  return <Trans>{count} inactive slots</Trans>;
}

function ManageButton({
  label,
  onOpen,
}: {
  label: string;
  onOpen: (button: HTMLButtonElement) => void;
}) {
  return (
    <Button
      className={manage}
      aria-label={label}
      aria-haspopup="dialog"
      onClick={(event) => onOpen(event.currentTarget)}
    >
      <span aria-hidden="true">⋮</span>
    </Button>
  );
}

/**
 * The state word of one slot. An unread or unclassified slot says `Unknown`,
 * never `Empty`: only a slot the backend proved to be completely empty may be
 * presented as one.
 */
function SlotStateLabel({ slot }: { slot: CharacterSlot | undefined }) {
  switch (slot?.state) {
    case "empty":
      return <Trans>Empty</Trans>;
    case "residual":
      return <Trans>Residual data</Trans>;
    default:
      return <Trans>Unknown</Trans>;
  }
}

/** `Slot N · Starting Class`, or `Slot N · Unknown` without a safe class. */
function SlotMeta({ characterID, startingClass }: { characterID: number; startingClass?: string }) {
  const slotNumber = characterID + 1;
  return startingClass === undefined ? (
    <Trans>Slot {slotNumber} · Unknown</Trans>
  ) : (
    <Trans>
      Slot {slotNumber} · {startingClass}
    </Trans>
  );
}

function ActiveRow({
  character,
  startingClass,
  selected,
  onSelect,
}: {
  character: CharacterSummary;
  startingClass?: string;
  selected: boolean;
  onSelect: () => void;
}) {
  const level = character.level;

  return (
    <Button className={row} pressed={selected} onClick={onSelect}>
      <span className={rowHead}>
        <span className={nameText}>{character.name}</span>
        <span className={levelText}>
          <Trans>RL {level}</Trans>
        </span>
      </span>
      <span className={meta}>
        <SlotMeta characterID={character.characterID} startingClass={startingClass} />
      </span>
    </Button>
  );
}

/**
 * The slot operations of one slot. Every action is capability-gated by the
 * backend projection, and the two destructive ones need a second confirmation
 * that names the slot and its consequence.
 *
 * A confirmation belongs to the exact context it was given in: the session, the
 * revision, the slot and the slot state. Any of them moving makes the standing
 * confirmation a confirmation of something else, so it is dropped instead of
 * carried over. The slot itself is re-read from the current list on every
 * render, and a slot that disappears from that list closes the dialog.
 *
 * The mutations are owned above this component, so its lifetime never releases
 * the in-flight lock.
 */
function SlotManagerDialog({
  management,
  model,
  mutations,
  characterID,
  token,
  startingClass,
  returnFocusRef,
  onClose,
}: {
  management: SlotManagement;
  model: CharacterSelection;
  mutations: CharacterSlotMutations;
  characterID: number;
  token: number;
  startingClass?: string;
  returnFocusRef: RefObject<HTMLElement | null>;
  onClose: (token: number) => void;
}) {
  const { t } = useLingui();
  const [confirmed, setConfirmed] = useState<string | undefined>(undefined);
  const [cloneTarget, setCloneTarget] = useState<string>("");

  const listed = model.characters.data;
  const slot = model.slotOf(characterID);
  const summary = listed?.characters.find((character) => character.characterID === characterID);
  const cloneTargets = model.slots.filter((candidate) => candidate.capabilities.cloneInto);
  const slotNumber = characterID + 1;

  const close = useCallback(() => onClose(token), [onClose, token]);

  /**
   * Whether the slots on screen were read for the session and revision every
   * operation would be sent with. A list read for another revision cannot
   * justify a request, so nothing is sent; the backend keeps its own
   * `expectedRevision` check behind this one.
   */
  const stated =
    listed !== undefined &&
    listed.saveSessionID === management.saveSessionID &&
    listed.saveRevision === management.saveRevision;

  // A committed operation whose refresh failed leaves the screen unable to state
  // anything about this slot, so it offers nothing more until the list is reloaded.
  const blocked = mutations.isBusy || management.sessionBusy === true || mutations.isDesynced;

  // The slot left the loaded session: the dialog lost its subject entirely.
  useEffect(() => {
    if (listed !== undefined && slot === undefined) close();
  }, [listed, slot, close]);

  const context =
    slot === undefined
      ? undefined
      : `${management.saveSessionID}:${management.saveRevision}:${characterID}:${slot.state}`;
  const confirming = confirmed !== undefined && confirmed === context;

  /** The capability as the current list reports it, re-read at click time. */
  const allows = (capability: keyof CharacterSlot["capabilities"]) =>
    model.slotOf(characterID)?.capabilities[capability] === true;

  const run = async (allowed: boolean, call: () => Promise<SlotMutationOutcome>) => {
    if (blocked || !stated || !allowed) return;
    const outcome = await call();
    // Only a request that changed nothing on screen leaves the dialog open, and
    // it never leaves a standing confirmation behind.
    if (outcome === "applied" || outcome === "unchanged") close();
    else setConfirmed(undefined);
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) close();
      }}
      title={t`Manage slot ${slotNumber}`}
      closeLabel={t`Close`}
      returnFocusRef={returnFocusRef}
    >
      <div className={dialogSection}>
        {slot === undefined ? (
          <p className={message}>
            <Trans>This slot is no longer part of the loaded session.</Trans>
          </p>
        ) : (
          <>
            <p className={meta}>
              {slot.state === "active" && summary !== undefined ? (
                <Trans>
                  {summary.name} — RL {summary.level}
                </Trans>
              ) : (
                <SlotMeta characterID={characterID} startingClass={startingClass} />
              )}
            </p>
            <p>
              <Badge tone={slot.state === "residual" ? "warning" : "neutral"}>
                <SlotStateBadge slot={slot} />
              </Badge>
            </p>

            {mutations.isDesynced && (
              <p role="alert" className={alertPanel}>
                <Trans>
                  The slot operation was applied, but the character list could not be refreshed. The
                  slots shown here are out of date and no further slot operation is offered until
                  the session catches up on its own.
                </Trans>
              </p>
            )}

            {!mutations.isDesynced && mutations.error !== undefined && (
              <p role="alert" className={alertPanel}>
                <Trans>The slot operation was rejected. Reload the list and try again.</Trans>
              </p>
            )}

            {slot.state === "residual" && (
              <p className={message}>
                <Trans>
                  This inactive slot still contains character data. Clearing it removes that data
                  from the session.
                </Trans>
              </p>
            )}

            {!confirming && !mutations.isDesynced && (
              <div className={dialogActions}>
                {slot.capabilities.activate && (
                  <Button
                    tone="accent"
                    disabled={blocked || !stated}
                    onClick={() =>
                      run(allows("activate"), () =>
                        mutations.setActive({
                          saveSessionID: management.saveSessionID,
                          characterID,
                          active: true,
                          expectedRevision: management.saveRevision,
                        }),
                      )
                    }
                  >
                    <Trans>Activate</Trans>
                  </Button>
                )}
                {slot.capabilities.deactivate && (
                  <Button
                    disabled={blocked || !stated}
                    onClick={() =>
                      run(allows("deactivate"), () =>
                        mutations.setActive({
                          saveSessionID: management.saveSessionID,
                          characterID,
                          active: false,
                          expectedRevision: management.saveRevision,
                        }),
                      )
                    }
                  >
                    <Trans>Deactivate</Trans>
                  </Button>
                )}
                {slot.capabilities.delete && (
                  <Button disabled={blocked || !stated} onClick={() => setConfirmed(context)}>
                    {slot.state === "residual" ? (
                      <Trans>Clean residual data</Trans>
                    ) : (
                      <Trans>Delete character</Trans>
                    )}
                  </Button>
                )}
              </div>
            )}

            {slot.capabilities.deactivate && !confirming && (
              <p className={message}>
                <Trans>Deactivating keeps the character data in the slot.</Trans>
              </p>
            )}

            {slot.capabilities.cloneFrom && !confirming && !mutations.isDesynced && (
              <div className={dialogSection}>
                <label htmlFor="clone-target-slot" className={meta}>
                  <Trans>Clone to slot…</Trans>
                </label>
                <Select
                  id="clone-target-slot"
                  value={cloneTarget}
                  disabled={blocked || cloneTargets.length === 0}
                  onChange={(event) => setCloneTarget(event.currentTarget.value)}
                >
                  <option value="">
                    {cloneTargets.length === 0 ? t`No empty slot` : t`Select a slot`}
                  </option>
                  {cloneTargets.map((candidate) => (
                    <option key={candidate.characterID} value={String(candidate.characterID)}>
                      {t`Slot ${candidate.characterID + 1}`}
                    </option>
                  ))}
                </Select>
                <Button
                  disabled={blocked || !stated || cloneTarget === ""}
                  onClick={() =>
                    run(
                      // The target must still be an offered destination now, not
                      // only when the list was drawn.
                      allows("cloneFrom") &&
                        cloneTargets.some(
                          (candidate) => String(candidate.characterID) === cloneTarget,
                        ),
                      () =>
                        mutations.clone({
                          saveSessionID: management.saveSessionID,
                          sourceCharacterID: characterID,
                          targetSlotID: Number(cloneTarget),
                          expectedRevision: management.saveRevision,
                        }),
                    )
                  }
                >
                  <Trans>Clone</Trans>
                </Button>
              </div>
            )}

            {confirming && (
              <div className={dialogSection}>
                <p role="alert" className={alertPanel}>
                  {slot.state === "residual" ? (
                    <Trans>
                      Clear the residual data of slot {slotNumber}? The remaining character data is
                      removed from the session.
                    </Trans>
                  ) : (
                    <Trans>
                      Delete the character in slot {slotNumber}? Its data is removed from the
                      session.
                    </Trans>
                  )}
                </p>
                <div className={dialogActions}>
                  <Button
                    disabled={blocked || !stated}
                    onClick={() =>
                      run(allows("delete"), () =>
                        mutations.remove({
                          saveSessionID: management.saveSessionID,
                          characterID,
                          expectedRevision: management.saveRevision,
                        }),
                      )
                    }
                  >
                    <Trans>Confirm</Trans>
                  </Button>
                  <Button disabled={blocked} onClick={() => setConfirmed(undefined)}>
                    <Trans>Cancel</Trans>
                  </Button>
                </div>
              </div>
            )}

            <p className={message}>
              <Trans>Slot operations stay session changes until the save is written.</Trans>
            </p>
          </>
        )}
      </div>
    </Dialog>
  );
}

function SlotStateBadge({ slot }: { slot: CharacterSlot }) {
  switch (slot.state) {
    case "active":
      return <Trans>Active</Trans>;
    case "empty":
      return <Trans>Empty</Trans>;
    case "residual":
      return <Trans>Residual data</Trans>;
    default:
      return <Trans>Unknown</Trans>;
  }
}
