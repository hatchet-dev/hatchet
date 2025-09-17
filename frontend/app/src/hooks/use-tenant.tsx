import { useCallback, useMemo, useState } from 'react';
import api, {
  UpdateTenantRequest,
  Tenant,
  CreateTenantRequest,
  queries,
} from '@/lib/api';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { BillingContext, lastTenantAtom } from '@/lib/atoms';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { Evaluate } from '@/lib/can/shared/permission.base';
import { useAtom } from 'jotai';

export type Plan = 'free' | 'starter' | 'growth';

export type EvaluateResult = {
  allowed: boolean;
  rejectReason?: RejectReason;
  message?: string;
};

export type PermissionSet<K = any> = Record<string, (resource?: K) => Evaluate>;

export enum RejectReason {
  BILLING_REQUIRED = 'BILLING_REQUIRED',
  UPGRADE_REQUIRED = 'UPGRADE_REQUIRED',
  ROLE_REQUIRED = 'ROLE_REQUIRED',
  CLOUD_ONLY = 'CLOUD_ONLY',
}

export function useCurrentTenantId() {
  const params = useParams();
  const tenantId = params.tenant;

  invariant(tenantId, 'Tenant ID is required');

  return { tenantId };
}

export function useTenantDetails() {
  const params = useParams();
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
  const location = useLocation();
  const navigate = useNavigate();

  const setTenant = useCallback(
    (tenant: Tenant) => {
      const currentPath = location.pathname;

      const newPath = currentPath.replace(
        /\/tenants\/([^/]+)/,
        `/tenants/${tenant.metadata.id}`,
      );

      setLastTenant(tenant);
      queryClient.clear();
      navigate(newPath);
    },
    [navigate, location.pathname, setLastTenant, queryClient],
  );

  const membership = useMemo(() => {
    if (!tenantId) {
      return undefined;
    }

    return memberships?.find(
      (membership) => membership.tenant?.metadata.id === tenantId,
    );
  }, [tenantId, memberships]);

  const tenant = membership?.tenant;

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

  const { data: cloudMeta } = useCloudApiMeta();

  const billingState = useQuery({
    ...queries.cloud.billing(tenant?.metadata?.id || ''),
    enabled: tenant && !!cloudMeta?.data.canBill,
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
    if (!cloudMeta?.data.canBill) {
      return;
    }

    return {
      state: billingState.data,
      setPollBilling,
      plan: subscriptionPlan,
      hasPaymentMethods,
    };
  }, [
    cloudMeta?.data.canBill,
    billingState.data,
    subscriptionPlan,
    hasPaymentMethods,
  ]);

  const can = useCallback(
    (evalFn: Evaluate) => {
      return evalFn({
        tenant,
        billing: billingContext,
        meta: cloudMeta?.data,
      });
    },
    [billingContext, cloudMeta?.data, tenant],
  );

  return {
    tenantId,
    tenant,
    isLoading: membershipsQuery.isLoading,
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
