import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { queries } from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

export const useWebhooks = (onDeleteSuccess?: () => void) => {
  const queryClient = useQueryClient();
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading, error } = useQuery({
    ...queries.v1Webhooks.list(tenantId),
  });

  const { mutate: deleteWebhook, isPending } = useMutation({
    mutationFn: async ({ webhookName }: { webhookName: string }) =>
      await api.v1WebhookDelete(tenantId, webhookName),
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

  return {
    data: data?.rows || [],
    isLoading,
    error,
    mutations: {
      deleteWebhook,
      isDeletePending: isPending,
    },
  };
};
