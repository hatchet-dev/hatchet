import { z } from 'zod';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  queries,
  V1CreateWebhookRequest,
  V1WebhookAuthType,
  V1WebhookHMACAlgorithm,
  V1WebhookHMACEncoding,
  V1WebhookSourceName,
} from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

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

  return {
    data: data?.rows || [],
    isLoading,
    error,
    mutations: {
      deleteWebhook,
      isDeletePending,
      createWebhook,
      isCreatePending,
    },
  };
};

export const webhookFormSchema = z.object({
  sourceName: z.nativeEnum(V1WebhookSourceName),
  name: z
    .string()
    .min(1, 'Name is required')
    .max(255, 'Name must be less than 255 characters'),
  eventKeyExpression: z.string().min(1, 'Event key expression is required'),
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
