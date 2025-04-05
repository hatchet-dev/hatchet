import { Tenant, TenantMemberRole } from '@/lib/api';
import useUser from './use-user';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

interface TenantState {
  tenant?: Tenant;
  role?: TenantMemberRole;
  isLoading: boolean;
  setTenant: (tenant: Tenant) => void;
}

export default function useTenant(): TenantState {
  const { memberships, isLoading: isUserLoading } = useUser();
  const [searchParams, setSearchParams] = useSearchParams();

  const setTenant = useCallback(
    (tenant: Tenant) => {
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.set('tenant', tenant.metadata.id);
      setSearchParams(newSearchParams, { replace: true });
    },
    [searchParams, setSearchParams],
  );

  const tenant = useMemo(() => {
    const tenantId =
      searchParams.get('tenant') || localStorage.getItem('tenant');

    if (tenantId == null) {
      const fallback = memberships?.[0];
      if (!fallback || !fallback.tenant) {
        return;
      }
      setTenant(fallback.tenant);
      return fallback;
    }

    const matched = memberships?.find(
      (membership) => membership.tenant?.metadata.id === tenantId,
    );

    if (!matched || !matched.tenant?.metadata.id) {
      return;
    }

    localStorage.setItem('tenant', matched.tenant?.metadata.id);

    return matched;
  }, [memberships, searchParams, setTenant]);

  return {
    tenant: tenant?.tenant,
    role: tenant?.role,
    isLoading: isUserLoading,
    setTenant,
  };
}
