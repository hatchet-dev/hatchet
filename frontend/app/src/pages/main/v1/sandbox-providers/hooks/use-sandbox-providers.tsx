import { labelsToRows, rowsToLabels } from '../../endpoints/lib/labels';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import {
  CreateSandboxProviderRequest,
  SandboxProvider,
  SandboxProviderKind,
  UpdateSandboxProviderRequest,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { z } from 'zod';

export const SANDBOX_PROVIDERS_REFETCH_INTERVAL_MS = 5000;

export const useSandboxProviders = () => {
  const { tenantId } = useCurrentTenantId();
  const { pagination, setPagination, setPageSize, limit, offset } =
    usePagination({ key: 'sandbox-providers' });

  const { data, isLoading, isPlaceholderData, error } = useQuery({
    ...queries.cloud.listSandboxProviders(tenantId, { offset, limit }),
    refetchInterval: SANDBOX_PROVIDERS_REFETCH_INTERVAL_MS,
    placeholderData: keepPreviousData,
  });

  return {
    data: data?.rows ?? [],
    pageCount: data?.pagination?.num_pages ?? 0,
    pagination,
    setPagination,
    setPageSize,
    isLoading,
    isPlaceholderData,
    error,
  };
};

export const useSandboxProviderMutations = () => {
  const queryClient = useQueryClient();
  const { tenantId } = useCurrentTenantId();

  const invalidate = async (providerId?: string) => {
    await queryClient.invalidateQueries({
      queryKey: queries.cloud.listSandboxProviders._def,
    });

    if (providerId) {
      await queryClient.invalidateQueries({
        queryKey: queries.cloud.getSandboxProvider(providerId).queryKey,
      });
      await queryClient.invalidateQueries({
        queryKey: queries.cloud.listSandboxProviderEvents(providerId).queryKey,
      });
    }
  };

  const create = useMutation({
    mutationFn: async (data: CreateSandboxProviderRequest) =>
      (await cloudApi.sandboxProviderCreate(tenantId, data)).data,
    onSuccess: async (provider) => invalidate(provider.metadata.id),
  });

  const update = useMutation({
    mutationFn: async ({
      providerId,
      data,
    }: {
      providerId: string;
      data: UpdateSandboxProviderRequest;
    }) => (await cloudApi.sandboxProviderUpdate(providerId, data)).data,
    onSuccess: async (provider) => invalidate(provider.metadata.id),
  });

  const remove = useMutation({
    mutationFn: async (providerId: string) =>
      (await cloudApi.sandboxProviderDelete(providerId)).data,
    onSuccess: async (provider) => {
      queryClient.removeQueries({
        queryKey: queries.cloud.getSandboxProvider(provider.metadata.id)
          .queryKey,
      });
      await invalidate();
    },
  });

  return { create, update, remove };
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
          message: 'Every label needs a key',
        });
        return;
      }

      if (k !== '' && seen.has(k)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Label "${k}" is set twice`,
        });
        return;
      }

      seen.add(k);
    }
  });

const configJson = z.string().superRefine((text, ctx) => {
  if (text.trim() === '') {
    return;
  }

  try {
    const parsed = JSON.parse(text);

    if (
      parsed === null ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed)
    ) {
      throw new Error('not an object');
    }
  } catch {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Config must be a JSON object',
    });
  }
});

const parseConfig = (text: string): Record<string, unknown> | undefined =>
  text.trim() === '' ? undefined : JSON.parse(text);

const sharedFields = {
  name: z
    .string()
    .min(1, 'Name is required')
    .max(255, 'Name must be at most 255 characters'),
  defaultTemplate: z.string(),
  maxIdleTtlSeconds: z
    .number({ invalid_type_error: 'Enter a number' })
    .int()
    .min(60, 'At least 60 seconds')
    .max(604800, 'At most 7 days'),
  configJson,
  labels: labelRows,
  enabled: z.boolean(),
};

export const providerCreateFormSchema = z.object({
  kind: z.nativeEnum(SandboxProviderKind),
  apiKey: z.string().min(1, 'API key is required'),
  ...sharedFields,
});

export type ProviderCreateFormData = z.infer<typeof providerCreateFormSchema>;

export const defaultCreateValues = (
  kind: SandboxProviderKind,
): ProviderCreateFormData => ({
  kind,
  name: '',
  apiKey: '',
  defaultTemplate: '',
  maxIdleTtlSeconds: 86400,
  configJson: '',
  labels: [],
  enabled: true,
});

export const buildCreateRequest = (
  data: ProviderCreateFormData,
): CreateSandboxProviderRequest => ({
  name: data.name.trim(),
  kind: data.kind,
  apiKey: data.apiKey,
  defaultTemplate: data.defaultTemplate.trim() || undefined,
  maxIdleTtlSeconds: data.maxIdleTtlSeconds,
  config: parseConfig(data.configJson),
  labels: rowsToLabels(data.labels),
  enabled: data.enabled,
});

export const providerUpdateFormSchema = z.object({
  apiKey: z.string(),
  ...sharedFields,
});

export type ProviderUpdateFormData = z.infer<typeof providerUpdateFormSchema>;

export const updateFormValues = (
  provider: SandboxProvider,
): ProviderUpdateFormData => ({
  name: provider.name,
  apiKey: '',
  defaultTemplate: provider.defaultTemplate ?? '',
  maxIdleTtlSeconds: provider.maxIdleTtlSeconds,
  configJson:
    Object.keys(provider.config ?? {}).length > 0
      ? JSON.stringify(provider.config, null, 2)
      : '',
  labels: labelsToRows(provider.labels),
  enabled: provider.enabled,
});

export const buildUpdateRequest = (
  data: ProviderUpdateFormData,
  provider: SandboxProvider,
): UpdateSandboxProviderRequest => ({
  name: data.name.trim(),
  apiKey: data.apiKey === '' ? undefined : data.apiKey,
  defaultTemplate: data.defaultTemplate.trim(),
  maxIdleTtlSeconds: data.maxIdleTtlSeconds,
  config: parseConfig(data.configJson) ?? {},
  labels: rowsToLabels(data.labels, provider.labels),
  enabled: data.enabled,
});
