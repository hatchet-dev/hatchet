import { TenantBillingState } from '@/next/lib/api/generated/cloud/data-contracts';
import { useQuery } from '@tanstack/react-query';
import useTenant from './use-tenant';
import useApiMeta from './use-api-meta';
import { cloudApi } from '@/next/lib/api/api';
import { useMemo } from 'react';

type Plan = 'free' | 'starter' | 'growth';

export type BillingHook = {
  data: {
    state: TenantBillingState | undefined;
    plan: Plan;
    hasPaymentMethods: boolean;
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
    queryKey: ['billing-state:get', tenant],
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

  return {
    data: {
      state: billingState,
      plan: subscriptionPlan,
      hasPaymentMethods,
    },
    isLoading,
    isError,
  };
}
