import { cloudApi } from '@/lib/api/api';
import { IdpInfoFromCustomer } from '@/lib/sso/sso-types';
import { extractSsoIdpInfo } from '@/lib/sso/sso-utils';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

export function ssoConfigQueryKey(orgId: string) {
  return ['sso-config', orgId] as const;
}

export function useSsoConfig(orgId: string) {
  const queryClient = useQueryClient();
  const queryKey = ssoConfigQueryKey(orgId);

  const query = useQuery({
    queryKey,
    queryFn: () => cloudApi.ssoGet(orgId),
    select: (response: Awaited<ReturnType<typeof cloudApi.ssoGet>>) =>
      extractSsoIdpInfo(response.data),
    staleTime: Infinity,
  });

  const upsertMutation = useMutation({
    mutationFn: async (idpInfo: IdpInfoFromCustomer) => {
      await cloudApi.ssoUpsert(orgId, { idpInfoFromCustomer: idpInfo });
      return idpInfo;
    },
    onSuccess: (idpInfo) => {
      queryClient.setQueryData(queryKey, idpInfo);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => cloudApi.ssoDelete(orgId),
    onSuccess: () => {
      queryClient.setQueryData(queryKey, null);
    },
  });

  return {
    idpConfiguration: query.data ?? null,
    loading: query.isLoading,
    upsertMutation,
    deleteMutation,
  };
}