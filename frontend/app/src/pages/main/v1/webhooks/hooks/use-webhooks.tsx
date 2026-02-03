import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  queries,
  V1CreateWebhookRequest,
  V1UpdateWebhookRequest,
  V1WebhookAuthType,
  V1WebhookHMACAlgorithm,
  V1WebhookHMACEncoding,
  V1WebhookSourceName,
} from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

export const useWebhooks = (onDeleteSuccess?: () => void) => {
  const queryClient = useQueryClient();
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading, error } = useQuery({
    ...queries.v1Webhooks.list(tenantId),
  });

  const { mutate: deleteWebhook, isPending: isDeletePending } = useMutation({
    mutationFn: async ({ webhookName }: { webhookName: string }) =>
      api.v1WebhookDelete(tenantId, webhookName),
    onSuccess: async () => {
      if (onDeleteSuccess) {
        onDeleteSuccess();
      }

      const queryKey = queries.v1Webhooks.list(tenantId).queryKey;
      await queryClient.invalidateQueries({
        queryKey,
      });
    },
  });

  const { mutate: createWebhook, isPending: isCreatePending } = useMutation({
    mutationFn: async (webhookData: V1CreateWebhookRequest) =>
      api.v1WebhookCreate(tenantId, webhookData),
    onSuccess: async () => {
      const queryKey = queries.v1Webhooks.list(tenantId).queryKey;
      await queryClient.invalidateQueries({
        queryKey,
      });
    },
  });

  const { mutate: updateWebhook, isPending: isUpdatePending } = useMutation({
    mutationFn: async ({
      webhookName,
      webhookData,
    }: {
      webhookName: string;
      webhookData: V1UpdateWebhookRequest;
    }) => api.v1WebhookUpdate(tenantId, webhookName, webhookData),
    onSuccess: async () => {
      const queryKey = queries.v1Webhooks.list(tenantId).queryKey;
      await queryClient.invalidateQueries({
        queryKey,
      });
    },
  });

  const createWebhookURL = (name: string) => {
    return `${window.location.protocol}//${window.location.hostname}/api/v1/stable/tenants/${tenantId}/webhooks/${name}`;
  };

  return {
    data: data?.rows || [],
    isLoading,
    error,
    createWebhookURL,
    mutations: {
      deleteWebhook,
      isDeletePending,
      createWebhook,
      isCreatePending,
      updateWebhook,
      isUpdatePending,
    },
  };
};

const optionalJsonString = z
  .string()
  .optional()
  .refine(
    (val) => {
      if (!val || val.trim() === '') return true;
      try {
        JSON.parse(val);
        return true;
      } catch {
        return false;
      }
    },
    { message: 'Must be valid JSON' },
  );

export const webhookFormSchema = z.object({
  sourceName: z.nativeEnum(V1WebhookSourceName),
  name: z.string().min(1, 'Name expression is required'),
  eventKeyExpression: z.string().min(1, 'Event key expression is required'),
  scopeExpression: z.string().optional(),
  staticPayload: optionalJsonString,
  authType: z.nativeEnum(V1WebhookAuthType),
  username: z.string().optional(),
  password: z.string().optional(),
  headerName: z.string().optional(),
  apiKey: z.string().optional(),
  signingSecret: z.string().optional(),
  algorithm: z.nativeEnum(V1WebhookHMACAlgorithm).optional(),
  encoding: z.nativeEnum(V1WebhookHMACEncoding).optional(),
  signatureHeaderName: z.string().optional(),
});

export type WebhookFormData = z.infer<typeof webhookFormSchema>;

export const webhookUpdateFormSchema = z.object({
  eventKeyExpression: z.string().min(1, 'Event key expression is required'),
  scopeExpression: z.string().optional(),
  staticPayload: optionalJsonString,
});

export type WebhookUpdateFormData = z.infer<typeof webhookUpdateFormSchema>;
