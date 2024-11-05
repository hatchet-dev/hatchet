import { atom, useAtom } from 'jotai';
import { Tenant, queries } from './api';
import { useSearchParams } from 'react-router-dom';
import { useEffect, useMemo, useState } from 'react';
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

// search param sets the tenant, the last tenant set is used if the search param is empty,
// otherwise the first membership is used
export function useTenantContext(): [
  Tenant | undefined,
  (tenant: Tenant) => void,
] {
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const [searchParams, setSearchParams] = useSearchParams();
  const [currTenant, setCurrTenant] = useState<Tenant>();

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const memberships = useMemo(() => {
    return listMembershipsQuery.data?.rows || [];
  }, [listMembershipsQuery]);

  const computedCurrTenant = useMemo(() => {
    const findTenant = (tenantId: string) => {
      return memberships?.find((m) => m.tenant?.metadata.id === tenantId)
        ?.tenant;
    };

    const currTenantId = searchParams.get('tenant') || undefined;

    if (currTenantId) {
      const tenant = findTenant(currTenantId);

      if (tenant) {
        return tenant;
      }
    }

    const lastTenantId = lastTenant?.metadata.id || undefined;

    if (lastTenantId) {
      const tenant = findTenant(lastTenantId);

      if (tenant) {
        return tenant;
      }
    }

    const firstMembershipTenant = memberships?.[0]?.tenant;

    return firstMembershipTenant;
  }, [memberships, lastTenant?.metadata.id, searchParams]);

  // sets the current tenant if the search param changes
  useEffect(() => {
    if (searchParams.get('tenant') !== currTenant?.metadata.id) {
      const newTenant = memberships?.find(
        (m) => m.tenant?.metadata.id === searchParams.get('tenant'),
      )?.tenant;

      if (newTenant) {
        setCurrTenant(newTenant);
      } else if (computedCurrTenant?.metadata.id) {
        const newSearchParams = new URLSearchParams(searchParams);
        newSearchParams.set('tenant', computedCurrTenant?.metadata.id);
        setSearchParams(newSearchParams, { replace: true });
      }
    }
  }, [
    searchParams,
    currTenant,
    setCurrTenant,
    memberships,
    computedCurrTenant,
    setSearchParams,
  ]);

  // sets the current tenant to the initial tenant
  useEffect(() => {
    if (!currTenant && computedCurrTenant) {
      setCurrTenant(computedCurrTenant);
    }
  }, [computedCurrTenant, currTenant, setCurrTenant]);

  // keeps the current tenant in sync with the last tenant
  useEffect(() => {
    if (currTenant && lastTenant?.metadata.id !== currTenant?.metadata.id) {
      setLastTenant(currTenant);
    }
  }, [lastTenant, currTenant, setLastTenant]);

  const setTenant = (tenant: Tenant) => {
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set('tenant', tenant.metadata.id);
    setSearchParams(newSearchParams, { replace: true });
  };

  return [currTenant || computedCurrTenant, setTenant];
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

type ViewOptions = 'graph' | 'minimap';

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
