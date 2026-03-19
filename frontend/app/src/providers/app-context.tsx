import { queries, Tenant, TenantMember, User } from '@/lib/api';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import type { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { lastTenantAtom } from '@/lib/atoms';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useAtom } from 'jotai';
import {
  createContext,
  useContext,
  useMemo,
  type ReactNode,
  useEffect,
} from 'react';

/**
 * Shared application context providing user, tenant, and organization data
 *
 * This provider consolidates the previously separate hooks:
 * - useCurrentUser
 * - useTenantDetails
 * - useOrganizations
 *
 * Pattern inspired by PostHogProvider - provides a single source of truth
 * for user, tenant, and organization data across the application.
 */

interface AppContextValue {
  // User data
  user: User | undefined;
  isUserLoading: boolean;
  userError: unknown;
  isUserError: boolean;

  // Tenant data
  tenant: Tenant | undefined;
  tenantId: string | undefined;
  isTenantLoading: boolean;
  membership: string | undefined;

  // Organization data (cloud only)
  organizations: OrganizationForUserList | undefined;
  isOrganizationsLoading: boolean;
  isCloudEnabled: boolean;
  isControlPlaneEnabled: boolean;

  // Helper to get organization for current tenant
  getCurrentOrganization: () =>
    | OrganizationForUserList['rows'][number]
    | undefined;
}

const AppContext = createContext<AppContextValue | null>(null);

interface AppContextProviderProps {
  children: ReactNode;
}

export function AppContextProvider({ children }: AppContextProviderProps) {
  const { isCloudEnabled, isCloudLoading } = useCloud();
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const orgApi = useOrganizationApi();

  // Get tenant ID from route params (following TanStack Router best practices)
  // This replaces the old useCurrentTenantId pattern
  const params = useParams({ strict: false });
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const tenantId = params.tenant || lastTenant?.metadata.id;

  // Fetch current user (routes to control plane when enabled via queries.user.current)
  const currentUserQuery = useQuery({
    ...queries.user.current,
    retry: false,
  });

  // Fetch tenant memberships
  const membershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  // Fetch organizations (cloud or control plane)
  const organizationsQuery = useQuery({
    queryKey: ['organization:list'],
    queryFn: async () => {
      const result = await orgApi.organizationList();
      return result.data;
    },
    enabled:
      (isCloudEnabled || isControlPlaneEnabled) &&
      !isCloudLoading &&
      !isControlPlaneLoading,
  });

  // Compute current membership and tenant
  const membership = useMemo(() => {
    if (!tenantId || !membershipsQuery.data?.rows) {
      return undefined;
    }

    return membershipsQuery.data.rows.find(
      (m: TenantMember) => m.tenant?.metadata.id === tenantId,
    );
  }, [tenantId, membershipsQuery.data?.rows]);

  const tenant = membership?.tenant;

  // Update last tenant atom when tenant changes
  useEffect(() => {
    if (tenant && tenant.metadata.id !== lastTenant?.metadata.id) {
      setLastTenant(tenant);
    }
  }, [tenant, lastTenant, setLastTenant]);

  // Helper to get organization for current tenant
  const getCurrentOrganization = useMemo(
    () => () => {
      if (!tenantId || !organizationsQuery.data?.rows) {
        return undefined;
      }

      return organizationsQuery.data.rows.find((org) =>
        (org.tenants || []).some((t) => t.id === tenantId),
      );
    },
    [tenantId, organizationsQuery.data?.rows],
  );

  const value = useMemo<AppContextValue>(
    () => ({
      // User
      user: currentUserQuery.data,
      isUserLoading: currentUserQuery.isLoading,
      userError: currentUserQuery.error,
      isUserError: currentUserQuery.isError,

      // Tenant
      tenant,
      tenantId,
      isTenantLoading: membershipsQuery.isLoading,
      membership: membership?.role,

      // Organizations
      organizations: organizationsQuery.data,
      isOrganizationsLoading: organizationsQuery.isLoading,
      isCloudEnabled,
      isControlPlaneEnabled,

      // Helpers
      getCurrentOrganization,
    }),
    [
      currentUserQuery.data,
      currentUserQuery.isLoading,
      currentUserQuery.error,
      currentUserQuery.isError,
      tenant,
      tenantId,
      membershipsQuery.isLoading,
      membership?.role,
      organizationsQuery.data,
      organizationsQuery.isLoading,
      isCloudEnabled,
      isControlPlaneEnabled,
      getCurrentOrganization,
    ],
  );

  useEffect(() => {
    console.log('[AppContext] state', {
      path: window.location.pathname,
      tenantId,
      tenantFromMembership: tenant?.metadata.id,
      membershipRole: membership?.role,
      lastTenantId: lastTenant?.metadata.id,
      isUserLoading: currentUserQuery.isLoading,
      hasUser: Boolean(currentUserQuery.data),
      membershipsCount: membershipsQuery.data?.rows?.length,
      isMembershipsLoading: membershipsQuery.isLoading,
      isCloudEnabled,
      isCloudLoading,
      isControlPlaneEnabled,
      isControlPlaneLoading,
      isOrganizationsLoading: organizationsQuery.isLoading,
      organizationsCount: organizationsQuery.data?.rows?.length,
    });
  }, [
    tenantId,
    tenant?.metadata.id,
    membership?.role,
    lastTenant?.metadata.id,
    currentUserQuery.isLoading,
    currentUserQuery.data,
    membershipsQuery.data?.rows?.length,
    membershipsQuery.isLoading,
    isCloudEnabled,
    isCloudLoading,
    isControlPlaneEnabled,
    isControlPlaneLoading,
    organizationsQuery.isLoading,
    organizationsQuery.data?.rows?.length,
  ]);

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

/**
 * Hook to access the shared app context
 *
 * Throws an error if used outside of AppContextProvider
 */
export function useAppContext() {
  const context = useContext(AppContext);

  if (!context) {
    throw new Error('useAppContext must be used within AppContextProvider');
  }

  return context;
}
