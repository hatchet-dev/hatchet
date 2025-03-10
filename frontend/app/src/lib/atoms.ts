import { atom, useAtom } from 'jotai';
import { Tenant, TenantVersion, queries } from './api';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

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

export const lastTenantAtom = atom(
  (get) => get(lastTenantAtomInit),
  (_get, set, newVal: Tenant) => {
    set(lastTenantAtomInit, newVal);
    localStorage.setItem(lastTenantKey, JSON.stringify(newVal));
  },
);

type TenantContextPresent = {
  tenant: Tenant;
  tenantId: string;
  setTenant: (tenant: Tenant) => void;
  setViewLegacyData: (val: boolean) => void;
};

type TenantContextMissing = {
  tenant: undefined;
  tenantId: undefined;
  setTenant: (tenant: Tenant) => void;
};

type TenantContext = TenantContextPresent | TenantContextMissing;

// search param sets the tenant, the last tenant set is used if the search param is empty,
// otherwise the first membership is used
export function useTenant(): TenantContext {
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const [searchParams, setSearchParams] = useSearchParams();

  const [viewLegacyData, setViewLegacyData] = useState(false);

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

  useEffect(() => {
    // Only redirect on initial tenant load
    if (lastRedirected == tenant?.slug) {
      return;
    }

    if (pathname.startsWith('/onboarding')) {
      return;
    }

    setLastRedirected(tenant?.slug);

    if (tenant?.version == TenantVersion.V0 && pathname.startsWith('/v1')) {
      return navigate({
        pathname: pathname.replace('/v1', ''),
        search: params.toString(),
      });
    }

    if (tenant?.version == TenantVersion.V1 && !pathname.startsWith('/v1')) {
      return navigate({
        pathname: '/v1' + pathname,
        search: params.toString(),
      });
    }
  }, [lastRedirected, navigate, params, pathname, tenant]);

  if (!tenant) {
    return {
      tenant: undefined,
      tenantId: undefined,
      setTenant,
    };
  }

  return {
    tenant,
    tenantId: tenant.metadata.id,
    setTenant,
    setViewLegacyData,
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
