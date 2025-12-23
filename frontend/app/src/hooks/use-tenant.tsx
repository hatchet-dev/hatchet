import api, {
  UpdateTenantRequest,
  Tenant,
  CreateTenantRequest,
  queries,
} from '@/lib/api';
import { BillingContext, lastTenantAtom } from '@/lib/atoms';
import { Evaluate } from '@/lib/can/shared/permission.base';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { appRoutes } from '@/router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMatchRoute, useNavigate, useParams } from '@tanstack/react-router';
import { useAtom } from 'jotai';
import { useCallback, useEffect, useMemo, useState } from 'react';

type Plan = 'free' | 'starter' | 'growth';

export function useCurrentTenantId() {
  const params = useParams({ from: appRoutes.tenantRoute.to });
  const tenantId = params.tenant;

  return { tenantId };
}

export function useTenantDetails() {
  // Allow calling this hook even when not currently on a tenant route
  // (e.g., onboarding pages). When not matched, params will be empty.
  const params = useParams({ strict: false });
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const tenantId = params.tenant || lastTenant?.metadata.id;

  const membershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const memberships = useMemo(
    () => membershipsQuery.data?.rows || [],
    [membershipsQuery.data],
  );

  const queryClient = useQueryClient();
  const matchRoute = useMatchRoute();
  const navigate = useNavigate();
  const tenantParamInPath = params.tenant;
  const [switchingTenantId, setSwitchingTenantId] = useState<string | null>(
    null,
  );

  const setTenant = useCallback(
    (tenant: Tenant) => {
      setSwitchingTenantId(tenant.metadata.id);
      setLastTenant(tenant);
      // On tenant-switch we want to avoid leaking tenant-scoped query data across tenants.
      // However, a full `queryClient.clear()` causes the top-nav switchers to briefly lose
      // their option/selected state (since memberships + user are global). Instead, we
      // remove everything except a small allowlist of global queries.
      queryClient.cancelQueries();
      queryClient.removeQueries({
        predicate: (query) => {
          const key0 = query.queryKey?.[0];

          // Global, safe-to-keep queries (not tenant-scoped):
          if (
            key0 === 'tenant-memberships:list' ||
            key0 === 'user:get' ||
            key0 === 'user:get:current' ||
            key0 === 'user:list-tenant-invites' ||
            key0 === 'user:list:tenant-invites' ||
            key0 === 'metadata' ||
            key0 === 'organization:list'
          ) {
            return false;
          }

          return true;
        },
      });

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

  // Clear the "switching" state once the router param reflects the newly-selected tenant.
  useEffect(() => {
    if (!switchingTenantId) {
      return;
    }

    if (params.tenant === switchingTenantId) {
      setSwitchingTenantId(null);
    }
  }, [params.tenant, switchingTenantId]);

  const membership = useMemo(() => {
    if (!tenantId) {
      return undefined;
    }

    return memberships?.find(
      (membership) => membership.tenant?.metadata.id === tenantId,
    );
  }, [tenantId, memberships]);

  const tenant =
    membership?.tenant ||
    (lastTenant?.metadata.id && lastTenant.metadata.id === tenantId
      ? lastTenant
      : undefined);

  // For UI (switchers) we want an optimistic, non-jumpy label during tenant transitions.
  // This does NOT change the effective tenant-scoped routing; it only stabilizes display.
  const displayTenant =
    switchingTenantId &&
    lastTenant?.metadata.id &&
    lastTenant.metadata.id === switchingTenantId
      ? lastTenant
      : tenant;

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
  });

  const subscriptionPlan: Plan = useMemo(() => {
    const plan = billingState.data?.subscription?.plan;
    if (!plan) {
      return 'free';
    }
    return plan as Plan;
  }, [billingState.data?.subscription?.plan]);

  const hasPaymentMethods = useMemo(() => {
    return (billingState.data?.paymentMethods?.length || 0) > 0;
  }, [billingState.data?.paymentMethods]);

  const billingContext: BillingContext | undefined = useMemo(() => {
    if (!cloud?.canBill) {
      return;
    }

    return {
      state: billingState.data,
      setPollBilling,
      plan: subscriptionPlan,
      hasPaymentMethods,
    };
  }, [cloud?.canBill, billingState.data, subscriptionPlan, hasPaymentMethods]);

  const can = useCallback(
    (evalFn: Evaluate) => {
      return evalFn({
        tenant,
        billing: billingContext,
        meta: cloud,
      });
    },
    [billingContext, cloud, tenant],
  );

  return {
    tenantId,
    tenant,
    displayTenant,
    isLoading: membershipsQuery.isLoading || membershipsQuery.isFetching,
    isSwitchingTenant: switchingTenantId !== null,
    membership: membership?.role,
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
