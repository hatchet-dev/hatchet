import api, {
  UpdateTenantRequest,
  Tenant,
  CreateTenantRequest,
  queries,
} from '@/lib/api';
import { BillingContext, lastTenantAtom } from '@/lib/atoms';
import { Evaluate } from '@/lib/can/shared/permission.base';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useAppContext } from '@/providers/app-context';
import { appRoutes } from '@/router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMatchRoute, useNavigate, useParams } from '@tanstack/react-router';
import { useAtom } from 'jotai';
import { useCallback, useMemo, useState } from 'react';

type Plan = 'free' | 'starter' | 'growth';

/**
 * Hook to get current tenant ID from route params
 *
 * @deprecated Prefer using route params directly via `useParams({ from: appRoutes.tenantRoute.to })`
 * This hook is maintained for backward compatibility during migration.
 */
export function useCurrentTenantId() {
  const params = useParams({ from: appRoutes.tenantRoute.to });
  const tenantId = params.tenant;

  return { tenantId };
}

/**
 * Hook for tenant details and operations
 *
 * Now backed by AppContext for better performance.
 * Gets tenant/membership data from context, but keeps all mutation logic here.
 */
export function useTenantDetails() {
  // Get tenant data from AppContext
  const appContext = useAppContext();
  const tenantId = appContext.tenantId;
  const tenant = appContext.tenant;
  const membership = appContext.membership;

  const queryClient = useQueryClient();
  const matchRoute = useMatchRoute();
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const [, setLastTenant] = useAtom(lastTenantAtom);
  const tenantParamInPath = params.tenant;

  const setTenant = useCallback(
    (tenant: Tenant) => {
      setLastTenant(tenant);
      queryClient.clear();

      const isOnTenantRoute = Boolean(
        matchRoute({
          to: appRoutes.tenantRoute.to,
          params: tenantParamInPath
            ? {
                tenant: tenantParamInPath,
              }
            : undefined,
          fuzzy: true,
        }),
      );

      if (!isOnTenantRoute) {
        navigate({
          to: appRoutes.tenantRunsRoute.to,
          params: { tenant: tenant.metadata.id },
        });
        return;
      }

      navigate({
        to: '.', // stay on the current route
        params: { tenant: tenant.metadata.id },
      });
    },
    [matchRoute, navigate, setLastTenant, queryClient, tenantParamInPath],
  );

  // Tenant and membership now come from AppContext
  // No need to compute them here anymore

  const createTenantMutation = useMutation({
    mutationKey: ['tenant:create'],
    mutationFn: async ({ name }: { name: string }): Promise<Tenant> => {
      const tenantData: CreateTenantRequest = {
        name,
        slug: name.toLowerCase().replace(/\s+/g, '-'),
      };

      const response = await api.tenantCreate(tenantData);
      return response.data;
    },
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ['user:*'] });
      if (data.metadata.id) {
        setTenant(data);
      }
      return data;
    },
  });

  const updateTenantMutation = useMutation({
    mutationKey: ['tenant:update', tenant?.metadata.id],
    mutationFn: async (data: UpdateTenantRequest) => {
      if (!tenant?.metadata.id) {
        throw new Error('Tenant not found');
      }
      const response = await api.tenantUpdate(tenant.metadata.id, data);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user:*'] });
      queryClient.invalidateQueries({ queryKey: ['tenant:*'] });
    },
  });

  const resourcePolicyQuery = useQuery({
    queryKey: ['tenant-resource-policy:get', tenant?.metadata.id],
    queryFn: async () => {
      return (await api.tenantResourcePolicyGet(tenant?.metadata.id ?? '')).data
        .limits;
    },
    enabled: !!tenant?.metadata.id,
  });

  const [pollBilling, setPollBilling] = useState(false);

  const { cloud, isCloudEnabled } = useCloud();

  const billingState = useQuery({
    ...queries.cloud.billing(tenant?.metadata?.id || ''),
    enabled: !!tenant?.metadata?.id && isCloudEnabled && !!cloud?.canBill,
    refetchInterval: pollBilling ? 1000 : false,
    retry: false,
  });

  const paymentMethodsQuery = useQuery({
    ...queries.cloud.paymentMethods(tenant?.metadata?.id || ''),
    enabled: !!tenant && !!cloud?.canBill,
    retry: false,
  });

  const subscriptionPlan: Plan = useMemo(() => {
    const plan = billingState.data?.currentSubscription?.plan;
    if (!plan) {
      return 'free';
    }
    return plan as Plan;
  }, [billingState.data?.currentSubscription?.plan]);

  const billingContext: BillingContext | undefined = useMemo(() => {
    if (!cloud?.canBill) {
      return;
    }

    const hasPaymentMethods = (paymentMethodsQuery.data?.length || 0) > 0;
    const isLoading = paymentMethodsQuery.isLoading || billingState.isLoading;

    return {
      state: billingState.data,
      setPollBilling,
      plan: subscriptionPlan,
      hasPaymentMethods,
      isLoading,
    };
  }, [
    cloud?.canBill,
    billingState.data,
    paymentMethodsQuery.data,
    paymentMethodsQuery.isLoading,
    billingState.isLoading,
    subscriptionPlan,
  ]);

  const can = useCallback(
    (evalFn: Evaluate) => {
      return evalFn({
        tenant,
        billing: billingContext,
        meta: cloud ?? undefined,
      });
    },
    [billingContext, cloud, tenant],
  );

  return {
    tenantId,
    tenant,
    isUserUniverseLoaded: appContext.isUserUniverseLoaded,
    membership,
    setTenant,
    create: createTenantMutation,
    update: {
      mutate: (data: UpdateTenantRequest) => updateTenantMutation.mutate(data),
      mutateAsync: (data: UpdateTenantRequest) =>
        updateTenantMutation.mutateAsync(data),
      isPending: updateTenantMutation.isPending,
    },
    limit: resourcePolicyQuery,
    billing: billingContext,
    can,
  };
}
