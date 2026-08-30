// The ONLY module allowed to import the generated Wails bindings. Everything
// above this layer depends on the application ports, never on generated code or
// on `window.go`.

import {
  CloseSave,
  GetApplicationInfo,
  GetCharacterProfile,
  GetCharacterStats,
  GetInventory,
  GetLoadedSave,
  GetSaveCharacters,
  GetStorage,
  LoadSave,
} from "../../../wailsjs/go/desktop/Bridge";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../../application/application-info/applicationInfoPort";
import type {
  CharacterPort,
  CharacterProfile,
  CharacterStats,
  SaveCharacters,
} from "../../application/character/characterPort";
import type { ItemPage, ItemsPort } from "../../application/items/itemsPort";
import type { SaveSession, SaveSessionPort } from "../../application/save-session/saveSessionPort";

/**
 * The stable code a failed bridge call is reported with. The transport error is
 * deliberately dropped here: a Wails rejection can carry a Go error string or a
 * runtime stack, and neither may reach the interface. The UI maps this code to
 * a localized message and never renders the code itself.
 *
 * The code says that the call failed and nothing more. Classifying a domain
 * failure would mean reading the rejection text, which is exactly what this
 * boundary refuses to do; a structured backend error contract is what a finer
 * distinction has to come from.
 */
export const bridgeFailureCode = "bridge_call_failed";

async function callBridge<T>(call: () => Promise<T>): Promise<T> {
  try {
    return await call();
  } catch {
    throw new Error(bridgeFailureCode);
  }
}

/** Projects the generated session result onto the application port shape. */
function toSaveSession(result: Awaited<ReturnType<typeof GetLoadedSave>>): SaveSession {
  return {
    saveSessionID: result.saveSessionID,
    platform: result.platform,
    format: result.format,
    unsavedChanges: result.unsavedChanges,
  };
}

/**
 * Projects a generated container page onto the application port shape. The two
 * generated result types carry the same fields, so one projection covers both
 * and neither container gets a second, drifting mapping.
 */
function toItemPage(
  result: Awaited<ReturnType<typeof GetInventory | typeof GetStorage>>,
): ItemPage {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    records: result.records.map((record) => ({
      ownedItemID: record.ownedItemID,
      kind: record.kind,
      key: record.key,
      gameID: record.gameID,
      containerSection: record.containerSection,
      physicalIndex: record.physicalIndex,
      gaItemHandle: record.gaItemHandle,
      quantity: record.quantity,
      acquisitionIndex: record.acquisitionIndex,
    })),
    total: result.total,
    page: result.page,
    pageSize: result.pageSize,
  };
}

/**
 * The single adapter behind every application port. A second, parallel
 * adaptation layer would give the generated bindings a second way into the
 * application, so all ports are fulfilled here.
 */
export const wailsDesktopBridge: ApplicationInfoPort & SaveSessionPort & CharacterPort & ItemsPort =
  {
    getApplicationInfo: async (): Promise<ApplicationInfo> => {
      const result = await callBridge(GetApplicationInfo);

      return {
        version: result.applicationVersion,
        schemas: result.supportedSchemas.map((schema) => ({
          name: schema.name,
          minimumVersion: schema.minimumVersion,
          currentVersion: schema.currentVersion,
        })),
        capabilities: [...result.capabilities],
      };
    },

    loadSave: async (source, expectedPlatform) =>
      toSaveSession(await callBridge(() => LoadSave(source, expectedPlatform))),

    getLoadedSave: async (saveSessionID) =>
      toSaveSession(await callBridge(() => GetLoadedSave(saveSessionID))),

    closeSave: async (saveSessionID) => {
      await callBridge(() => CloseSave(saveSessionID));
    },

    getSaveCharacters: async (saveSessionID): Promise<SaveCharacters> => {
      const result = await callBridge(() => GetSaveCharacters(saveSessionID));

      return {
        saveSessionID: result.saveSessionID,
        characters: result.characters.map((summary) => ({
          characterID: summary.characterID,
          active: summary.active,
          name: summary.name,
          level: summary.level,
        })),
      };
    },

    getCharacterProfile: async (saveSessionID, characterID): Promise<CharacterProfile> => {
      const result = await callBridge(() => GetCharacterProfile(saveSessionID, characterID));

      return {
        saveSessionID: result.saveSessionID,
        characterID: result.characterID,
        active: result.active,
        name: result.name,
        level: result.level,
        startingClassID: result.startingClassID,
        gender: result.gender,
        secondsPlayed: result.secondsPlayed,
      };
    },

    getCharacterStats: async (saveSessionID, characterID): Promise<CharacterStats> => {
      const result = await callBridge(() => GetCharacterStats(saveSessionID, characterID));

      return {
        saveSessionID: result.saveSessionID,
        characterID: result.characterID,
        active: result.active,
        vigor: result.vigor,
        mind: result.mind,
        endurance: result.endurance,
        strength: result.strength,
        dexterity: result.dexterity,
        intelligence: result.intelligence,
        faith: result.faith,
        arcane: result.arcane,
        level: result.level,
        hp: result.hp,
        maxHP: result.maxHP,
        baseMaxHP: result.baseMaxHP,
        fp: result.fp,
        maxFP: result.maxFP,
        baseMaxFP: result.baseMaxFP,
        sp: result.sp,
        maxSP: result.maxSP,
        baseMaxSP: result.baseMaxSP,
      };
    },

    // The five arguments reach the bridge in the order the backend contract
    // defines; the grouped request only protects the caller from transposing them.
    getInventory: async ({ saveSessionID, characterID, containerSection, page, pageSize }) =>
      toItemPage(
        await callBridge(() =>
          GetInventory(saveSessionID, characterID, containerSection, page, pageSize),
        ),
      ),

    getStorage: async ({ saveSessionID, characterID, containerSection, page, pageSize }) =>
      toItemPage(
        await callBridge(() =>
          GetStorage(saveSessionID, characterID, containerSection, page, pageSize),
        ),
      ),
  };
