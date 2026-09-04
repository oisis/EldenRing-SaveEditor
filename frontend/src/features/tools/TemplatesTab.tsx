import { Trans, useLingui } from "@lingui/react/macro";
import { useState } from "react";
import type { MutationReceipt } from "../../application/save-session/saveSessionPort";
import {
  useApplyBuildTemplate,
  useBuildTemplatePreview,
  useBuildTemplates,
  useTemplateLibraryMutations,
} from "../../application/templates/useBuildTemplates";
import type {
  BuildTemplateOverrides,
  BuildTemplateSummary,
} from "../../application/templates/templatePort";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Checkbox } from "../../ui/components/Checkbox/Checkbox";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { Table, TableFrame } from "../../ui/components/Table/Table";
import { actionCell, alert, message, tableFrame } from "../../ui/patterns/panel.css";
import {
  actionsBar,
  badgeRow,
  formGrid,
  sections,
  settingField,
  settingsRow,
} from "./ToolsPanel.css";

/** The page size of the library table. One page is plenty for a local library. */
const pageSize = 50;

/**
 * The four item apply modes and the four layout modes the backend accepts. They
 * are the backend's own vocabulary; the interface only words them.
 */
const itemModes = ["addMissing", "updateExisting", "merge", "replace"] as const;
const layoutModes = ["ignore", "append", "reorderOnly", "replace"] as const;

export type TemplatesTabProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<unknown>;
  sessionBusy: boolean;
};

/**
 * `Tools → Templates`.
 *
 * Every column, filter and override below comes from a confirmed backend
 * contract. The risk filter section 4.10.2 mentions is deliberately absent:
 * the Build Template library carries no risk level, and computing one here
 * would be a frontend rule standing in for a contract that does not exist.
 */
export function TemplatesTab({
  saveSessionID,
  saveRevision,
  characterID,
  applyMutationReceipt,
  sessionBusy,
}: TemplatesTabProps) {
  const { t } = useLingui();
  const [search, setSearch] = useState("");
  const [tagFilter, setTagFilter] = useState("");
  const [selected, setSelected] = useState<BuildTemplateSummary>();
  const [overrides, setOverrides] = useState<BuildTemplateOverrides>({});
  const [pendingDelete, setPendingDelete] = useState<BuildTemplateSummary>();
  const [editing, setEditing] = useState<BuildTemplateSummary>();
  const [creating, setCreating] = useState(false);
  const [syncFailed, setSyncFailed] = useState(false);

  const tags = tagFilter === "" ? [] : [tagFilter];
  const library = useBuildTemplates({ search, tags, page: 0, pageSize });
  const preview = useBuildTemplatePreview({
    saveSessionID,
    characterID,
    saveRevision,
    templateID: selected?.templateID,
    overrides,
  });
  const apply = useApplyBuildTemplate();
  const mutations = useTemplateLibraryMutations();

  // Only the tags the library actually reports can be filtered on, so the
  // control never offers a value that would return nothing.
  const availableTags = [
    ...new Set((library.data?.templates ?? []).flatMap((template) => template.tags)),
  ].sort();

  const sessionReady =
    saveSessionID !== undefined && saveRevision !== undefined && characterID !== undefined;
  const applyDisabled =
    !sessionReady ||
    sessionBusy ||
    apply.isPending ||
    preview.data?.executable !== true ||
    preview.data.saveRevision !== saveRevision;

  async function applySelected() {
    if (applyDisabled || selected === undefined || !sessionReady) return;
    setSyncFailed(false);
    let receipt: MutationReceipt;
    try {
      receipt = await apply.mutateAsync({
        saveSessionID,
        characterID,
        templateID: selected.templateID,
        expectedRevision: saveRevision,
        overrides,
      });
    } catch {
      // The mutation's own error state below already reports this; it is caught
      // so it never escapes unhandled.
      return;
    }
    try {
      await applyMutationReceipt(receipt);
    } catch {
      setSyncFailed(true);
    }
  }

  return (
    <div className={sections}>
      <Card aria-label={t`Build Templates`} className={sections}>
        <h2>
          <Trans>Build Templates</Trans>
        </h2>

        <div className={settingsRow}>
          <span className={settingField}>
            <label htmlFor="templates-search">
              <Trans>Search</Trans>
            </label>
            <Input
              id="templates-search"
              type="search"
              value={search}
              onChange={(event) => setSearch(event.currentTarget.value)}
            />
          </span>
          <span className={settingField}>
            <label htmlFor="templates-tag">
              <Trans>Tag</Trans>
            </label>
            <Select
              id="templates-tag"
              value={tagFilter}
              onChange={(event) => setTagFilter(event.currentTarget.value)}
            >
              <option value="">{t`All tags`}</option>
              {availableTags.map((tag) => (
                <option key={tag} value={tag}>
                  {tag}
                </option>
              ))}
            </Select>
          </span>
        </div>

        <div className={actionsBar}>
          <Button
            size="sm"
            disabled={mutations.importTemplate.isPending}
            onClick={() => mutations.importTemplate.mutate()}
          >
            <Trans>Import a template file</Trans>
          </Button>
          <Button size="sm" disabled={!sessionReady || sessionBusy} onClick={() => setCreating(true)}>
            <Trans>Create from the active character</Trans>
          </Button>
        </div>

        {/* The risk filter of the accepted mockup is stated as unavailable
            rather than synthesised: no backend field carries a template risk
            level, so there is nothing honest to filter on. */}
        <p className={message}>
          <Trans>
            A risk filter is not offered: the backend reports no risk level for a template.
          </Trans>
        </p>

        {library.isError ? (
          <p role="alert" className={alert}>
            <Trans>The template library could not be read.</Trans>
          </p>
        ) : null}

        <TableFrame className={tableFrame}>
          <Table>
            <caption>{t`Build Templates`}</caption>
            <thead>
              <tr>
                <th scope="col">{t`Name`}</th>
                <th scope="col">{t`Tags`}</th>
                <th scope="col">{t`Sections`}</th>
                <th scope="col">{t`Items`}</th>
                <th scope="col">{t`Updated`}</th>
                <th scope="col" className={actionCell}>
                  {t`Actions`}
                </th>
              </tr>
            </thead>
            <tbody>
              {(library.data?.templates ?? []).map((template) => (
                <tr key={template.templateID}>
                  <th scope="row">{template.name}</th>
                  <td>
                    <span className={badgeRow}>
                      {template.tags.map((tag) => (
                        <Badge key={tag}>{tag}</Badge>
                      ))}
                    </span>
                  </td>
                  <td>{template.selectedSections.join(", ")}</td>
                  <td>{template.inventoryItems + template.storageItems}</td>
                  <td>{template.updatedAt}</td>
                  <td className={actionCell}>
                    <span className={actionsBar}>
                      <Button
                        size="sm"
                        pressed={selected?.templateID === template.templateID}
                        onClick={() => {
                          setSelected(template);
                          setSyncFailed(false);
                        }}
                      >
                        <Trans>Preview</Trans>
                      </Button>
                      <Button size="sm" onClick={() => setEditing(template)}>
                        <Trans>Edit</Trans>
                      </Button>
                      <Button size="sm" onClick={() => setPendingDelete(template)}>
                        <Trans>Delete</Trans>
                      </Button>
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </TableFrame>
        {library.isSuccess && library.data.templates.length === 0 ? (
          <p className={message}>
            <Trans>This library has no templates yet.</Trans>
          </p>
        ) : null}
      </Card>

      {selected !== undefined ? (
        <Card aria-label={t`Template preview`} className={sections}>
          <h2>
            <Trans>Preview</Trans>: {selected.name}
          </h2>

          <div className={formGrid}>
            <span className={settingField}>
              <label htmlFor="templates-items-mode">
                <Trans>Items</Trans>
              </label>
              <Select
                id="templates-items-mode"
                value={overrides.itemsMode ?? ""}
                onChange={(event) =>
                  setOverrides((current) => ({
                    ...current,
                    itemsMode:
                      event.currentTarget.value === "" ? undefined : event.currentTarget.value,
                  }))
                }
              >
                <option value="">{t`Use the template's own option`}</option>
                {itemModes.map((mode) => (
                  <option key={mode} value={mode}>
                    {mode}
                  </option>
                ))}
              </Select>
            </span>
            <span className={settingField}>
              <label htmlFor="templates-inventory-layout">
                <Trans>Inventory layout</Trans>
              </label>
              <Select
                id="templates-inventory-layout"
                value={overrides.inventoryLayoutMode ?? ""}
                onChange={(event) =>
                  setOverrides((current) => ({
                    ...current,
                    inventoryLayoutMode:
                      event.currentTarget.value === "" ? undefined : event.currentTarget.value,
                  }))
                }
              >
                <option value="">{t`Use the template's own option`}</option>
                {layoutModes.map((mode) => (
                  <option key={mode} value={mode}>
                    {mode}
                  </option>
                ))}
              </Select>
            </span>
            <span className={settingField}>
              <label htmlFor="templates-storage-layout">
                <Trans>Storage layout</Trans>
              </label>
              <Select
                id="templates-storage-layout"
                value={overrides.storageLayoutMode ?? ""}
                onChange={(event) =>
                  setOverrides((current) => ({
                    ...current,
                    storageLayoutMode:
                      event.currentTarget.value === "" ? undefined : event.currentTarget.value,
                  }))
                }
              >
                <option value="">{t`Use the template's own option`}</option>
                {layoutModes.map((mode) => (
                  <option key={mode} value={mode}>
                    {mode}
                  </option>
                ))}
              </Select>
            </span>
            <span className={settingField}>
              <span className={badgeRow}>
                <Checkbox
                  id="templates-weapon-levels"
                  checked={overrides.useTemplateWeaponLevels ?? true}
                  onChange={(event) =>
                    setOverrides((current) => ({
                      ...current,
                      useTemplateWeaponLevels: event.currentTarget.checked,
                    }))
                  }
                />
                <label htmlFor="templates-weapon-levels">
                  <Trans>Use the template's own weapon upgrade levels</Trans>
                </label>
              </span>
            </span>
          </div>

          {!sessionReady ? (
            <p className={message}>
              <Trans>Open a save and select a character to preview a template.</Trans>
            </p>
          ) : null}
          {preview.isPending && sessionReady ? (
            <p role="status" className={message}>
              <Trans>Building the preview…</Trans>
            </p>
          ) : null}
          {preview.isError ? (
            <p role="alert" className={alert}>
              <Trans>The preview could not be built.</Trans>
            </p>
          ) : null}
          {preview.isSuccess ? (
            <>
              {preview.data.plan.profile?.level?.changed ? (
                <p className={message}>
                  <Trans>Level</Trans>: {preview.data.plan.profile.level.current} →{" "}
                  {preview.data.plan.profile.level.target}
                </p>
              ) : null}
              {preview.data.plan.stats !== undefined ? (
                <ul className={badgeRow}>
                  {preview.data.plan.stats.fields
                    .filter((entry) => entry.change.changed)
                    .map((entry) => (
                      <li key={entry.field}>
                        <Badge mono>
                          {`${entry.field} ${entry.change.current} → ${entry.change.target}`}
                        </Badge>
                      </li>
                    ))}
                </ul>
              ) : null}
              {preview.data.plan.spells !== undefined ? (
                <p className={message}>
                  <Trans>Spell slots changed</Trans>: {preview.data.plan.spells.changedSlots} (
                  {preview.data.plan.spells.usedMemorySlots}/
                  {preview.data.plan.spells.availableMemorySlots})
                </p>
              ) : null}
              {preview.data.blockingIssues.map((issue) => (
                <p key={`${issue.code}-${issue.section ?? ""}-${issue.field ?? ""}`} role="alert" className={alert}>
                  {/* The backend's own issue message: it names the exact
                      rejected section and field and carries no private data. */}
                  {issue.message}
                </p>
              ))}
            </>
          ) : null}

          <div className={actionsBar}>
            <Button tone="accent" disabled={applyDisabled} onClick={() => void applySelected()}>
              <Trans>Apply template</Trans>
            </Button>
            <Button onClick={() => setSelected(undefined)}>
              <Trans>Close preview</Trans>
            </Button>
          </div>
          {apply.isError ? (
            <p role="alert" className={alert}>
              <Trans>The template was not applied and nothing was changed.</Trans>
            </p>
          ) : null}
          {syncFailed ? (
            <p role="alert" className={alert}>
              <Trans>
                The template was applied, but this screen could not be refreshed. Reopen the save
                to see its current state.
              </Trans>
            </p>
          ) : null}
        </Card>
      ) : null}

      <TemplateMetadataDialog
        open={editing !== undefined}
        title={t`Edit template`}
        initialName={editing?.name ?? ""}
        initialDescription={editing?.description ?? ""}
        initialTags={(editing?.tags ?? []).join(", ")}
        busy={mutations.update.isPending}
        failed={mutations.update.isError}
        onSubmit={({ name, description, tags }) => {
          if (editing === undefined) return;
          mutations.update.mutate(
            {
              templateID: editing.templateID,
              templateRevision: editing.templateRevision,
              name,
              description,
              tags,
            },
            { onSuccess: () => setEditing(undefined) },
          );
        }}
        onClose={() => setEditing(undefined)}
      />

      <TemplateMetadataDialog
        open={creating}
        title={t`Create a template from the active character`}
        initialName=""
        initialDescription=""
        initialTags=""
        busy={mutations.create.isPending}
        failed={mutations.create.isError}
        onSubmit={({ name, description, tags }) => {
          if (!sessionReady) return;
          mutations.create.mutate(
            {
              saveSessionID,
              sourceCharacterID: characterID,
              name,
              description,
              tags,
            },
            { onSuccess: () => setCreating(false) },
          );
        }}
        onClose={() => setCreating(false)}
      />

      <Dialog
        open={pendingDelete !== undefined}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(undefined);
        }}
        title={t`Delete this template?`}
        description={t`The template is removed from this computer's library. No save is changed.`}
        closeLabel={t`Cancel`}
      >
        <p className={message}>{pendingDelete?.name}</p>
        <Button
          disabled={mutations.remove.isPending}
          onClick={() => {
            if (pendingDelete === undefined) return;
            mutations.remove.mutate(
              {
                templateID: pendingDelete.templateID,
                templateRevision: pendingDelete.templateRevision,
              },
              {
                onSuccess: () => {
                  if (selected?.templateID === pendingDelete.templateID) setSelected(undefined);
                  setPendingDelete(undefined);
                },
              },
            );
          }}
        >
          <Trans>Delete the template</Trans>
        </Button>
        {mutations.remove.isError ? (
          <p role="alert" className={alert}>
            <Trans>The template was not deleted.</Trans>
          </p>
        ) : null}
      </Dialog>
    </div>
  );
}
/**
 * The shared name, description and tags form of creating and editing a
 * template. The draft is remounted with the dialog it belongs to, so a value
 * typed for one template can never be submitted against another.
 */
function TemplateMetadataDialog({
  open,
  title,
  initialName,
  initialDescription,
  initialTags,
  busy,
  failed,
  onSubmit,
  onClose,
}: {
  open: boolean;
  title: string;
  initialName: string;
  initialDescription: string;
  initialTags: string;
  busy: boolean;
  failed: boolean;
  onSubmit: (values: { name: string; description: string; tags: string[] }) => void;
  onClose: () => void;
}) {
  const { t } = useLingui();
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={title}
      closeLabel={t`Cancel`}
    >
      {open ? (
        <TemplateMetadataForm
          key={`${title}:${initialName}`}
          initialName={initialName}
          initialDescription={initialDescription}
          initialTags={initialTags}
          busy={busy}
          failed={failed}
          onSubmit={onSubmit}
        />
      ) : null}
    </Dialog>
  );
}

function TemplateMetadataForm({
  initialName,
  initialDescription,
  initialTags,
  busy,
  failed,
  onSubmit,
}: {
  initialName: string;
  initialDescription: string;
  initialTags: string;
  busy: boolean;
  failed: boolean;
  onSubmit: (values: { name: string; description: string; tags: string[] }) => void;
}) {
  const [name, setName] = useState(initialName);
  const [description, setDescription] = useState(initialDescription);
  const [tags, setTags] = useState(initialTags);

  return (
    <div className={sections}>
      <span className={settingField}>
        <label htmlFor="template-form-name">
          <Trans>Name</Trans>
        </label>
        <Input
          id="template-form-name"
          value={name}
          onChange={(event) => setName(event.currentTarget.value)}
        />
      </span>
      <span className={settingField}>
        <label htmlFor="template-form-description">
          <Trans>Description</Trans>
        </label>
        <Input
          id="template-form-description"
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
        />
      </span>
      <span className={settingField}>
        <label htmlFor="template-form-tags">
          <Trans>Tags, separated by commas</Trans>
        </label>
        <Input
          id="template-form-tags"
          value={tags}
          onChange={(event) => setTags(event.currentTarget.value)}
        />
      </span>
      <Button
        tone="accent"
        disabled={busy || name.trim() === ""}
        onClick={() =>
          onSubmit({
            name: name.trim(),
            description: description.trim(),
            tags: tags
              .split(",")
              .map((tag) => tag.trim())
              .filter((tag) => tag !== ""),
          })
        }
      >
        <Trans>Save the template</Trans>
      </Button>
      {failed ? (
        <p role="alert" className={alert}>
          <Trans>The template was not stored.</Trans>
        </p>
      ) : null}
    </div>
  );
}
