import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { useTemplatePort } from "./templateClient";
import type {
  BuildTemplateOverrides,
  BuildTemplatePage,
  BuildTemplatePreview,
  TemplateApplyRequest,
  TemplateCreateRequest,
  TemplateMutationReceipt,
} from "./templatePort";

/**
 * The overrides rendered as one stable cache-key fragment. They are part of the
 * preview key because a different set of overrides is a different plan.
 */
function overridesKey(overrides: BuildTemplateOverrides | undefined): string {
  return overrides === undefined ? "" : JSON.stringify(overrides);
}

/** One page of the local Build Templates library. */
export function useBuildTemplates(request: {
  search: string;
  tags: readonly string[];
  page: number;
  pageSize: number;
}) {
  const port = useTemplatePort();

  return useQuery<BuildTemplatePage>({
    queryKey: queryKeys.buildTemplates(
      request.search,
      request.tags,
      request.page,
      request.pageSize,
    ),
    queryFn: () => port.getBuildTemplates(request),
    retry: false,
  });
}

/**
 * The preview of one template against the open character.
 *
 * It runs only with a session, a character and a revision, which is what makes
 * a preview a statement about one exact save state. Selecting nothing keeps the
 * query disabled rather than sending a request the backend would reject.
 */
export function useBuildTemplatePreview(request: {
  saveSessionID: string | undefined;
  characterID: number | undefined;
  saveRevision: string | undefined;
  templateID: string | undefined;
  overrides?: BuildTemplateOverrides | undefined;
}) {
  const port = useTemplatePort();
  const enabled =
    request.saveSessionID !== undefined &&
    request.characterID !== undefined &&
    request.saveRevision !== undefined &&
    request.templateID !== undefined;

  return useQuery<BuildTemplatePreview>({
    queryKey: queryKeys.buildTemplatePreview(
      request.saveSessionID ?? "",
      request.characterID ?? noCharacter,
      request.saveRevision ?? "",
      request.templateID ?? "",
      overridesKey(request.overrides),
    ),
    queryFn: () =>
      port.getBuildTemplatePreview({
        // The four values are proven present by `enabled`; a disabled query
        // never runs this function.
        saveSessionID: request.saveSessionID as string,
        characterID: request.characterID as number,
        templateID: request.templateID as string,
        overrides: request.overrides,
      }),
    enabled,
    retry: false,
  });
}

/**
 * Applies one template.
 *
 * The result is the shared save mutation receipt, so the caller routes it
 * through the same session refresh as every other save mutation instead of
 * inventing a second one here.
 */
export function useApplyBuildTemplate() {
  const port = useTemplatePort();

  return useMutation<TemplateMutationReceipt, Error, TemplateApplyRequest>({
    mutationFn: (request) => port.applyBuildTemplate(request),
  });
}

/**
 * The four library mutations. They change the local template library and not a
 * save, so they invalidate the template keys only and never a session key.
 */
export function useTemplateLibraryMutations() {
  const port = useTemplatePort();
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["templates"] });

  const create = useMutation<{ templateID: string }, Error, TemplateCreateRequest>({
    mutationFn: (request) => port.createBuildTemplate(request),
    onSuccess: invalidate,
  });
  const update = useMutation<
    { templateID: string },
    Error,
    {
      templateID: string;
      templateRevision: string;
      name: string;
      description?: string | undefined;
      tags?: readonly string[] | undefined;
    }
  >({
    mutationFn: (request) => port.updateBuildTemplate(request),
    onSuccess: invalidate,
  });
  const remove = useMutation<
    { templateID: string },
    Error,
    { templateID: string; templateRevision: string }
  >({
    mutationFn: (request) => port.deleteBuildTemplate(request),
    onSuccess: invalidate,
  });
  const importTemplate = useMutation<{ templateID?: string | undefined }, Error, void>({
    mutationFn: () => port.importBuildTemplate(),
    onSuccess: invalidate,
  });

  return { create, update, remove, importTemplate };
}
