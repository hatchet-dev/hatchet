import { t } from '../components/copy';
import {
  MIN_SIGNING_SECRET_LENGTH,
  PLATFORMS,
  deriveEndpointUrls,
  validateEndpointUrl,
} from '../lib/platforms';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  queries,
  V1CreateServerlessEndpointRequest,
  V1ServerlessEndpointKind,
  V1UpdateServerlessEndpointRequest,
} from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

export const ENDPOINTS_REFETCH_INTERVAL_MS = 5000;

export const useEndpoints = () => {
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading, error } = useQuery({
    ...queries.v1ServerlessEndpoints.list(tenantId),
    refetchInterval: ENDPOINTS_REFETCH_INTERVAL_MS,
  });

  return { data: data?.rows || [], isLoading, error };
};

/**
 * Mutations on serverless endpoints. Every mutation refreshes the list and the
 * endpoint's own queries, so list and detail pages stay in step.
 */
export const useEndpointMutations = () => {
  const queryClient = useQueryClient();
  const { tenantId } = useCurrentTenantId();

  const invalidate = async (endpointId?: string) => {
    await queryClient.invalidateQueries({
      queryKey: queries.v1ServerlessEndpoints.list(tenantId).queryKey,
    });

    if (endpointId) {
      await queryClient.invalidateQueries({
        queryKey: queries.v1ServerlessEndpoints.get(endpointId).queryKey,
      });
      await queryClient.invalidateQueries({
        queryKey: queries.v1ServerlessEndpoints.events(endpointId).queryKey,
      });
    }
  };

  const create = useMutation({
    mutationFn: async (data: V1CreateServerlessEndpointRequest) =>
      (await api.v1ServerlessEndpointCreate(tenantId, data)).data,
    onSuccess: async (endpoint) => invalidate(endpoint.metadata.id),
  });

  const update = useMutation({
    mutationFn: async ({
      endpointId,
      data,
    }: {
      endpointId: string;
      data: V1UpdateServerlessEndpointRequest;
    }) => (await api.v1ServerlessEndpointUpdate(endpointId, data)).data,
    onSuccess: async (endpoint) => invalidate(endpoint.metadata.id),
  });

  const remove = useMutation({
    mutationFn: async (endpointId: string) =>
      (await api.v1ServerlessEndpointDelete(endpointId)).data,
    onSuccess: async (endpoint) => {
      queryClient.removeQueries({
        queryKey: queries.v1ServerlessEndpoints.get(endpoint.metadata.id)
          .queryKey,
      });
      await invalidate();
    },
  });

  return { create, update, remove };
};

export type LabelRow = { key: string; value: string };

// whole numbers, without leading zeros, are saved as integer worker labels
const INTEGER_LABEL = /^-?(0|[1-9]\d*)$/;

export const labelsToRows = (
  labels: Record<string, unknown> | undefined,
): LabelRow[] =>
  Object.entries(labels ?? {}).map(([key, value]) => ({
    key,
    value: String(value),
  }));

export const rowsToLabels = (
  rows: LabelRow[],
): Record<string, string | number> => {
  const labels: Record<string, string | number> = {};

  for (const { key, value } of rows) {
    if (key.trim() === '' && value.trim() === '') {
      continue;
    }

    labels[key.trim()] = INTEGER_LABEL.test(value.trim())
      ? Number(value.trim())
      : value;
  }

  return labels;
};

const labelRows = z
  .array(z.object({ key: z.string(), value: z.string() }))
  .superRefine((rows, ctx) => {
    const seen = new Set<string>();

    for (const { key, value } of rows) {
      const k = key.trim();

      if (k === '' && value.trim() !== '') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: t('validation.labelKeyRequired'),
        });
        return;
      }

      if (k !== '' && seen.has(k)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: t('validation.labelKeyDuplicate', { key: k }),
        });
        return;
      }

      seen.add(k);
    }
  });

const numberField = () =>
  z.number({ invalid_type_error: t('validation.number') }).int();

const advancedFields = {
  requestTimeoutSeconds: numberField()
    .min(1, t('validation.requestTimeoutMin'))
    .max(600, t('validation.requestTimeoutMax')),
  pollIntervalSeconds: numberField().min(1, t('validation.pollIntervalMin')),
  inlineWaitBudgetMs: numberField().min(0, t('validation.inlineWaitMin')),
  labels: labelRows,
  enabled: z.boolean(),
};

const urlIssue = (ctx: z.RefinementCtx, path: string, value: string) => {
  const problem = validateEndpointUrl(value);

  if (problem) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: [path],
      message: t(problem),
    });
  }
};

export const endpointCreateFormSchema = z
  .object({
    kind: z.nativeEnum(V1ServerlessEndpointKind),
    name: z
      .string()
      .min(1, t('validation.nameRequired'))
      .max(255, t('validation.nameTooLong')),
    baseUrl: z.string(),
    // generic endpoints always set the URLs themselves
    manualUrls: z.boolean(),
    healthcheckUrl: z.string(),
    triggerUrl: z.string(),
    signingSecret: z
      .string()
      .min(
        MIN_SIGNING_SECRET_LENGTH,
        t('validation.secretTooShort', { length: MIN_SIGNING_SECRET_LENGTH }),
      ),
    ...advancedFields,
  })
  .superRefine((data, ctx) => {
    if (usesDerivedUrls(data)) {
      urlIssue(ctx, 'baseUrl', data.baseUrl);
      return;
    }

    urlIssue(ctx, 'healthcheckUrl', data.healthcheckUrl);
    urlIssue(ctx, 'triggerUrl', data.triggerUrl);
  });

export type EndpointCreateFormData = z.infer<typeof endpointCreateFormSchema>;

export const usesDerivedUrls = (data: {
  kind: V1ServerlessEndpointKind;
  manualUrls: boolean;
}) => !data.manualUrls && !!PLATFORMS[data.kind].healthcheckPath;

export const resolveCreateUrls = (data: EndpointCreateFormData) =>
  usesDerivedUrls(data)
    ? deriveEndpointUrls(data.kind, data.baseUrl)
    : {
        healthcheckUrl: data.healthcheckUrl.trim(),
        triggerUrl: data.triggerUrl.trim(),
      };

export const buildCreateRequest = (
  data: EndpointCreateFormData,
): V1CreateServerlessEndpointRequest => ({
  name: data.name.trim(),
  kind: data.kind,
  ...resolveCreateUrls(data),
  signingSecret: data.signingSecret,
  requestTimeoutSeconds: data.requestTimeoutSeconds,
  pollIntervalSeconds: data.pollIntervalSeconds,
  inlineWaitBudgetMs: data.inlineWaitBudgetMs,
  labels: rowsToLabels(data.labels),
  enabled: data.enabled,
});

export const endpointUpdateFormSchema = z
  .object({
    kind: z.nativeEnum(V1ServerlessEndpointKind),
    name: z
      .string()
      .min(1, t('validation.nameRequired'))
      .max(255, t('validation.nameTooLong')),
    healthcheckUrl: z.string(),
    triggerUrl: z.string(),
    // empty keeps the current secret
    signingSecret: z
      .string()
      .refine(
        (val) => val === '' || val.length >= MIN_SIGNING_SECRET_LENGTH,
        t('validation.secretTooShort', { length: MIN_SIGNING_SECRET_LENGTH }),
      ),
    ...advancedFields,
  })
  .superRefine((data, ctx) => {
    urlIssue(ctx, 'healthcheckUrl', data.healthcheckUrl);
    urlIssue(ctx, 'triggerUrl', data.triggerUrl);
  });

export type EndpointUpdateFormData = z.infer<typeof endpointUpdateFormSchema>;

export const buildUpdateRequest = (
  data: EndpointUpdateFormData,
): V1UpdateServerlessEndpointRequest => ({
  name: data.name.trim(),
  kind: data.kind,
  healthcheckUrl: data.healthcheckUrl.trim(),
  triggerUrl: data.triggerUrl.trim(),
  signingSecret: data.signingSecret === '' ? undefined : data.signingSecret,
  requestTimeoutSeconds: data.requestTimeoutSeconds,
  pollIntervalSeconds: data.pollIntervalSeconds,
  inlineWaitBudgetMs: data.inlineWaitBudgetMs,
  // labels replace the existing set, so removing every row clears them
  labels: rowsToLabels(data.labels),
  enabled: data.enabled,
});
