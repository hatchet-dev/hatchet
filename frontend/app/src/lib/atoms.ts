import { atom, useAtom } from 'jotai';
import { Tenant, TenantVersion, queries } from './api';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { TenantBillingState } from './api/generated/cloud/data-contracts';
import { Evaluate } from './can/shared/permission.base';

const getInitialValue = <T>(key: string, defaultValue?: T): T | undefined => {
  const item = localStorage.getItem(key);

  if (item !== null) {
    return JSON.parse(item) as T;
  }

  if (defaultValue !== undefined) {
    return defaultValue;
  }

  return;
};

const lastTenantKey = 'lastTenant';

const lastTenantAtomInit = atom(getInitialValue<Tenant>(lastTenantKey));

const lastTenantAtom = atom(
  (get) => get(lastTenantAtomInit),
  (_get, set, newVal: Tenant) => {
    set(lastTenantAtomInit, newVal);
    localStorage.setItem(lastTenantKey, JSON.stringify(newVal));
  },
);

type Plan = 'free' | 'starter' | 'growth';

export type BillingContext = {
  state: TenantBillingState | undefined;
  setPollBilling: (pollBilling: boolean) => void;
  plan: Plan;
  hasPaymentMethods: boolean;
};

type Can = (evalFn: Evaluate) => ReturnType<Evaluate>;

type TenantContextPresent = {
  tenant: Tenant;
  tenantId: string;
  setTenant: (tenant: Tenant) => void;
  billing?: BillingContext;
  can: Can;
};

type TenantContextMissing = {
  tenant: undefined;
  tenantId: undefined;
  setTenant: (tenant: Tenant) => void;
  billing: undefined;
  can: Can;
};

type TenantContext = TenantContextPresent | TenantContextMissing;

// search param sets the tenant, the last tenant set is used if the search param is empty,
// otherwise the first membership is used
export function useTenant(): TenantContext {
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const [searchParams, setSearchParams] = useSearchParams();

  const setTenant = useCallback(
    (tenant: Tenant) => {
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.set('tenant', tenant.metadata.id);
      setSearchParams(newSearchParams, { replace: true });
      setLastTenant(tenant);
    },
    [searchParams, setSearchParams, setLastTenant],
  );

  const membershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const memberships = useMemo(
    () => membershipsQuery.data?.rows || [],
    [membershipsQuery.data],
  );

  const findTenant = useCallback(
    (tenantId: string) => {
      return memberships?.find((m) => m.tenant?.metadata.id === tenantId)
        ?.tenant;
    },
    [memberships],
  );

  const computedCurrTenant = useMemo(() => {
    const currTenantId = searchParams.get('tenant') || undefined;
    const lastTenantId = lastTenant?.metadata.id || undefined;

    // If the current tenant is set as a query param, use it
    if (currTenantId) {
      const tenant = findTenant(currTenantId);

      if (tenant) {
        return tenant;
      }
    }

    // Otherwise, if a tenant was set in Jotai, use that as a fallback
    if (lastTenantId) {
      const tenant = findTenant(lastTenantId);

      if (tenant) {
        return tenant;
      }
    }

    // Finally, if neither a current tenant is set as a query param
    // nor if a tenant was set in Jotai, use the first membership as a fallback
    const firstMembershipTenant = memberships.at(0)?.tenant;

    return firstMembershipTenant;
  }, [memberships, searchParams, findTenant, lastTenant]);

  const currTenantId = searchParams.get('tenant');
  const currTenant = currTenantId ? findTenant(currTenantId) : undefined;

  const tenant = currTenant || computedCurrTenant;

  // If the tenant is not set as a query param at any point,
  // set it.
  // NOTE: This is helpful mostly for debugging to easily grab
  // the tenant from the URL.
  useEffect(() => {
    const currentTenantParam = searchParams.get('tenant');
    if (!currentTenantParam && tenant) {
      setTenant(tenant);
    }
  }, [searchParams, tenant, setTenant]);

  // Set the correct path for tenant version
  // NOTE: this is hacky and not ideal

  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const [lastRedirected, setLastRedirected] = useState<string | undefined>();
  const [previewV0, setPreviewV0] = useState<boolean>(false);

  useEffect(() => {
    const previewV0Params = params.get('previewV0');

    if (previewV0Params == 'false' && previewV0) {
      setPreviewV0(false);
    } else if (previewV0Params == 'true' && !previewV0) {
      setPreviewV0(true);
      return;
    }

    if (previewV0) {
      return;
    }

    if (pathname.startsWith('/onboarding')) {
      return;
    }

    if (tenant?.version == TenantVersion.V0 && pathname.startsWith('/v1')) {
      setLastRedirected(tenant?.slug);
      return navigate({
        pathname: pathname.replace('/v1', ''),
        search: params.toString(),
      });
    }

    if (tenant?.version == TenantVersion.V1 && !pathname.startsWith('/v1')) {
      setLastRedirected(tenant?.slug);
      return navigate({
        pathname: '/v1' + pathname,
        search: params.toString(),
      });
    }
  }, [lastRedirected, navigate, params, pathname, previewV0, tenant]);

  // Tenant Billing State

  const [pollBilling, setPollBilling] = useState(false);

  const cloudMeta = useCloudApiMeta();

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

  if (!tenant) {
    return {
      tenant: undefined,
      tenantId: undefined,
      setTenant,
      billing: undefined,
      can,
    };
  }

  return {
    tenant,
    tenantId: tenant.metadata.id,
    setTenant,
    billing: billingContext,
    can,
  };
}

const lastTimeRange = 'lastTimeRange';

const lastTimeRangeAtomInit = atom(
  getInitialValue<string>(lastTimeRange, '1h'),
);

export const lastTimeRangeAtom = atom(
  (get) => get(lastTimeRangeAtomInit),
  (_get, set, newVal: string) => {
    set(lastTimeRangeAtomInit, newVal);
    localStorage.setItem(lastTimeRange, JSON.stringify(newVal));
  },
);

const lastWorkerMetricsTimeRange = 'lastWorkerMetricsTimeRange';

const lastWorkerMetricsTimeRangeAtomInit = atom(
  getInitialValue<string>(lastWorkerMetricsTimeRange, '1h'),
);

export const lastWorkerMetricsTimeRangeAtom = atom(
  (get) => get(lastWorkerMetricsTimeRangeAtomInit),
  (_get, set, newVal: string) => {
    set(lastWorkerMetricsTimeRangeAtomInit, newVal);
    localStorage.setItem(lastWorkerMetricsTimeRange, JSON.stringify(newVal));
  },
);

export type ViewOptions = 'graph' | 'minimap';

const preferredWorkflowRunViewKey = 'wrView';

const preferredWorkflowRunViewAtomInit = atom(
  getInitialValue<ViewOptions>(preferredWorkflowRunViewKey, 'minimap'),
);

export const preferredWorkflowRunViewAtom = atom(
  (get) => get(preferredWorkflowRunViewAtomInit),
  (_get, set, newVal: ViewOptions) => {
    set(preferredWorkflowRunViewAtomInit, newVal);
    localStorage.setItem(preferredWorkflowRunViewKey, JSON.stringify(newVal));
  },
);
