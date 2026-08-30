import { Trans, useLingui } from "@lingui/react/macro";
import type { CharacterSummary } from "../../application/character/characterPort";
import { Button } from "../../ui/components/Button/Button";
import {
  alert,
  group,
  groupTitle,
  inactiveRow,
  level as levelText,
  list,
  message,
  meta,
  name as nameText,
  row,
  rowHead,
  sidebar,
} from "./CharacterSidebar.css";
import type { CharacterSelection } from "./useCharacterSelection";

/**
 * The read-only character panel. It renders exactly what the current backend
 * getters report: a name, a rune level and the slot number. It shows no
 * starting class, no play time, no file name, no slot status beyond `active`
 * and no raw identifier, because no getter reports them today.
 *
 * The panel is presentational: the selection controller is passed in, so the
 * screen that owns the session also owns the queries.
 */
export function CharacterSidebar({ model }: { model: CharacterSelection }) {
  const { t } = useLingui();
  const { hasSession, characters, activeCharacters, inactiveCharacters } = model;

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
        <p role="alert" className={alert}>
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
              <li key={character.characterID}>
                <ActiveRow
                  character={character}
                  selected={model.selectedCharacterID === character.characterID}
                  onSelect={() => model.selectCharacter(character.characterID)}
                />
              </li>
            ))}
          </ul>
        </section>
      )}

      {inactiveCharacters.length > 0 && (
        <section className={group}>
          <h3 className={groupTitle}>
            <Trans>Inactive slots</Trans>
          </h3>
          <ul className={list}>
            {inactiveCharacters.map((character) => (
              // Not a control: an inactive slot carries no read-only action, and
              // the backend does not tell `Empty` from `Residual data`, so the
              // row states only what is known.
              <li key={character.characterID} className={inactiveRow}>
                <span className={nameText}>
                  <Trans>Inactive slot</Trans>
                </span>
                <span className={meta}>{slotLabel(character.characterID)}</span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </aside>
  );
}

function ActiveRow({
  character,
  selected,
  onSelect,
}: {
  character: CharacterSummary;
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
      <span className={meta}>{slotLabel(character.characterID)}</span>
    </Button>
  );
}

/**
 * The user-facing slot number. The backend slot index is zero based and is
 * never shown; the panel always presents `Slot 1` upwards.
 */
function slotLabel(characterID: number) {
  const slotNumber = characterID + 1;
  return <Trans>Slot {slotNumber}</Trans>;
}
