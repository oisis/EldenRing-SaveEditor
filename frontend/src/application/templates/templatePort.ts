/**
 * The port the Templates module needs.
 *
 * Every field mirrors a confirmed backend contract. Nothing here invents a risk
 * level, a rating or a category the Build Template library does not carry: the
 * interface can only offer what the backend actually reports.
 */

import type { MutationReceipt } from "../save-session/saveSessionPort";

/** One template of the library index. */
export type BuildTemplateSummary = {
  templateID: string;
  name: string;
  description?: string | undefined;
  tags: readonly string[];
  createdAt: string;
  updatedAt: string;
  schemaVersion?: number | undefined;
  /** The sections the template carries, as the backend named them. */
  selectedSections: readonly string[];
  inventoryItems: number;
  storageItems: number;
  warnings: number;
  templateRevision: string;
};

export type BuildTemplatePage = {
  templates: readonly BuildTemplateSummary[];
  total: number;
  page: number;
  pageSize: number;
};

/** One numeric field of a preview plan. */
export type PreviewNumberChange = {
  current: number;
  target: number;
  changed: boolean;
};

export type PreviewTextChange = {
  current: string;
  target: string;
  changed: boolean;
};

/**
 * The preview plan, carried exactly as the backend built it. The three sections
 * are optional because a template need not carry all of them.
 */
export type BuildTemplatePlan = {
  profile?:
    | {
        name?: PreviewTextChange | undefined;
        level?: PreviewNumberChange | undefined;
      }
    | undefined;
  stats?:
    | {
        fields: readonly { field: string; change: PreviewNumberChange }[];
        resultLevel: number;
        resultSoulMemory: number;
      }
    | undefined;
  spells?:
    | {
        changedSlots: number;
        usedMemorySlots: number;
        availableMemorySlots: number;
      }
    | undefined;
};

export type BuildTemplateIssue = {
  code: string;
  section?: string | undefined;
  field?: string | undefined;
  message: string;
};

export type BuildTemplatePreview = {
  templateID: string;
  templateRevision: string;
  characterID: number;
  saveSessionID: string;
  saveRevision: string;
  executable: boolean;
  plan: BuildTemplatePlan;
  blockingIssues: readonly BuildTemplateIssue[];
};

/**
 * The overrides the backend accepts with a preview and an apply. They are the
 * confirmed apply options and nothing more: the section selection is not
 * offered, because its backend contract carries no serialisable shape the
 * frontend could state.
 */
export type BuildTemplateOverrides = {
  /** "addMissing", "updateExisting", "merge" or "replace". */
  itemsMode?: string | undefined;
  preserveExtraItems?: boolean | undefined;
  /** "ignore", "append", "reorderOnly" or "replace". */
  inventoryLayoutMode?: string | undefined;
  storageLayoutMode?: string | undefined;
  useTemplateWeaponLevels?: boolean | undefined;
  standardUpgradeOverride?: number | undefined;
  somberUpgradeOverride?: number | undefined;
};

/**
 * The shared save mutation receipt an apply produces. It is the save session's
 * own type, not a copy: applying a template refreshes the session through
 * exactly the same path as every other save mutation.
 */
export type TemplateMutationReceipt = MutationReceipt;

export type TemplatePreviewRequest = {
  saveSessionID: string;
  characterID: number;
  templateID: string;
  overrides?: BuildTemplateOverrides | undefined;
};

export type TemplateApplyRequest = TemplatePreviewRequest & {
  expectedRevision: string;
};

export type TemplateCreateRequest = {
  saveSessionID: string;
  sourceCharacterID: number;
  name: string;
  description?: string | undefined;
  tags?: readonly string[] | undefined;
};

export type TemplatePort = {
  getBuildTemplates: (request: {
    search: string;
    tags: readonly string[];
    page: number;
    pageSize: number;
  }) => Promise<BuildTemplatePage>;
  getBuildTemplatePreview: (request: TemplatePreviewRequest) => Promise<BuildTemplatePreview>;
  applyBuildTemplate: (request: TemplateApplyRequest) => Promise<TemplateMutationReceipt>;
  createBuildTemplate: (request: TemplateCreateRequest) => Promise<{ templateID: string }>;
  updateBuildTemplate: (request: {
    templateID: string;
    templateRevision: string;
    name: string;
    description?: string | undefined;
    tags?: readonly string[] | undefined;
  }) => Promise<{ templateID: string }>;
  deleteBuildTemplate: (request: {
    templateID: string;
    templateRevision: string;
  }) => Promise<{ templateID: string }>;
  /**
   * Opens the native document dialog and imports the chosen file. Cancelling
   * returns an undefined identifier and is not an error. The import is local:
   * no URL can be stated and no network call is made.
   */
  importBuildTemplate: () => Promise<{ templateID?: string | undefined }>;
};
