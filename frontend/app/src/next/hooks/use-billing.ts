import { TenantBillingState } from '@/lib/api/generated/cloud/data-contracts';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  UseQueryResult,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import useApiMeta from './use-api-meta';
import { cloudApi } from '@/lib/api/api';
import { useMemo } from 'react';
import queryClient from '@/query-client';

export type Plan = 'free' | 'starter' | 'growth';

export type BillingHook = {
  billing: {
    state: TenantBillingState | undefined;
    plan: Plan;
    hasPaymentMethods: boolean;
    getManagedUrl: UseQueryResult<string | undefined, Error>;
    changePlan: UseMutationResult<void, Error, { plan_code: Plan }>;
  };
  isLoading: boolean;
  isError: boolean;
};

interface UseBillingOptions {
  refetchInterval?: number;
}

export default function useBilling({
  refetchInterval,
}: UseBillingOptions = {}): BillingHook {
  const meta = useApiMeta();

  const { tenant } = useTenant();

  const {
    data: billingState,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ['billing:state:get', tenant?.metadata.id],
    queryFn: async () =>
      (await cloudApi.tenantBillingStateGet(tenant?.metadata.id || '')).data,
    enabled: meta.cloud?.canBill,
    refetchInterval,
  });

  const subscriptionPlan: Plan = useMemo(() => {
    if (!billingState?.subscription?.plan) {
      return 'free';
    }
    return billingState.subscription.plan as Plan;
  }, [billingState]);

  const hasPaymentMethods = useMemo(() => {
    return (billingState?.paymentMethods?.length || 0) > 0;
  }, [billingState]);

  const getManagedUrl = useQuery({
    queryKey: ['billing:get-managed-url', tenant?.metadata.id],
    queryFn: async () => {
      const response = await cloudApi.billingPortalLinkGet(
        tenant?.metadata.id || '',
      );
      return response.data.url;
    },
    enabled: !!tenant?.metadata.id,
  });

  const subscriptionMutation = useMutation({
    mutationKey: ['billing:subscription:update'],
    mutationFn: async ({ plan_code }: { plan_code: Plan }) => {
      const [plan, period] = plan_code.split(':');
      await cloudApi.subscriptionUpsert(tenant?.metadata.id || '', {
        plan,
        period,
      });
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['tenant:*'],
        }),
        queryClient.invalidateQueries({
          queryKey: ['billing:*'],
        }),
      ]);
    },
  });

  return {
    billing: {
      state: billingState,
      plan: subscriptionPlan,
      hasPaymentMethods,
      getManagedUrl,
      changePlan: subscriptionMutation,
    },
    isLoading,
    isError,
  };
}
