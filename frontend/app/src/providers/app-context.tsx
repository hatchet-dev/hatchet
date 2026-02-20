import { queries, Tenant, User } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import type { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import assert from '@/lib/assert';
import { lastTenantAtom } from '@/lib/atoms';
import useCloud from '@/pages/auth/hooks/use-cloud';
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

export type AppContextValue = {
  // User data
  user: User | undefined;
  isUserLoading: boolean;
  isUserLoaded: boolean;
  userError: unknown;
  isUserError: boolean;

  // Tenant data
  tenant: Tenant | undefined;
  tenantId: string | undefined;
  isTenantLoading: boolean;
  membership: string | undefined;
  refetchTenantMemberships: () => void;

  refetchOrganizations: () => void;
  isCloudEnabled: boolean;

  // Helper to get organization for current tenant
  getCurrentOrganization: () =>
    | OrganizationForUserList['rows'][number]
    | undefined;
} & OrganizationPossibilties;

type OrganizationPossibilties = {
  isOrganizationsLoading: boolean;
} & (
  | {
      isCloudEnabled: true;
      organizationsAreLoaded: true;
      organizations: OrganizationForUserList['rows'];
    }
  | {
      isCloudEnabled: boolean;
      organizationsAreLoaded: false;
      organizations: null;
    }
);

const AppContext = createContext<AppContextValue | null>(null);

interface AppContextProviderProps {
  children: ReactNode;
}

export function AppContextProvider({ children }: AppContextProviderProps) {
  const { isCloudEnabled } = useCloud();

  // Get tenant ID from route params (following TanStack Router best practices)
  // This replaces the old useCurrentTenantId pattern
  const params = useParams({ strict: false });
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const tenantId = params.tenant || lastTenant?.metadata.id;

  // Fetch current user
  const currentUserQuery = useQuery({
    ...queries.user.current,
    retry: false,
  });

  // Fetch tenant memberships
  const membershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  // Fetch organizations (cloud only)
  const organizationsQuery = useQuery({
    queryKey: ['organization:list'],
    queryFn: async () => {
      const result = await cloudApi.organizationList();
      return result.data;
    },
    enabled: isCloudEnabled,
  });

  const organizations = organizationsQuery.isSuccess
    ? organizationsQuery.data?.rows || []
    : null;

  // Compute current membership and tenant
  const membership = useMemo(() => {
    if (!tenantId || !membershipsQuery.data?.rows) {
      return undefined;
    }

    return membershipsQuery.data.rows.find(
      (m) => m.tenant?.metadata.id === tenantId,
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

  const organizationsAreLoaded = organizationsQuery.isSuccess;

  const organizationPossibilties: OrganizationPossibilties = useMemo(() => {
    const isOrganizationsLoading = organizationsQuery.isLoading;
    if (organizationsAreLoaded) {
      assert(organizations && isCloudEnabled);
      return {
        isOrganizationsLoading,
        isCloudEnabled,
        organizationsAreLoaded,
        organizations,
      };
    }

    return {
      isOrganizationsLoading,
      isCloudEnabled,
      organizationsAreLoaded: false,
      organizations: null,
    };
  }, [organizationsAreLoaded, organizations, isCloudEnabled]);

  const value = useMemo<AppContextValue>(
    () => ({
      // User
      user: currentUserQuery.data,
      isUserLoaded: currentUserQuery.isSuccess,
      isUserLoading: currentUserQuery.isLoading,
      userError: currentUserQuery.error,
      isUserError: currentUserQuery.isError,

      // Tenant
      tenant,
      tenantId,
      isTenantLoading: membershipsQuery.isLoading,
      membership: membership?.role,
      refetchTenantMemberships: membershipsQuery.refetch,

      refetchOrganizations: organizationsQuery.refetch,
      ...organizationPossibilties,

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
      getCurrentOrganization,
    ],
  );

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
