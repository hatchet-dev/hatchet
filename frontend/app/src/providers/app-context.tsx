import { useUserUniverse } from './user-universe';
import { queries, Tenant, User } from '@/lib/api';
import type { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { lastTenantAtom } from '@/lib/atoms';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useAtom } from 'jotai';
import {
  createContext,
  useContext,
  useMemo,
  type ReactNode,
  useEffect,
  useCallback,
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

type UserUniverseData =
  | {
      isUserUniverseLoaded: false;
      membership: undefined;
      organizations: undefined;
    }
  | {
      isUserUniverseLoaded: true;
      isCloudEnabled: true;
      membership: string | undefined;
      organizations: OrganizationForUserList['rows'];
    }
  | {
      isUserUniverseLoaded: true;
      isCloudEnabled: false;
      membership: string | undefined;
      organizations: undefined;
    };

export type AppContextValue = {
  // User data
  user: User | undefined;
  isUserLoading: boolean;
  isUserLoaded: boolean;
  userError: unknown;
  isUserError: boolean;
  invalidateCurrentUser: () => void;

  // Tenant data
  tenant: Tenant | undefined;
  tenantId: string | undefined;
} & UserUniverseData;

const AppContext = createContext<AppContextValue | null>(null);

interface AppContextProviderProps {
  children: ReactNode;
}

export function AppContextProvider({ children }: AppContextProviderProps) {
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

  const queryClient = useQueryClient();

  const invalidateCurrentUser = useCallback(() => {
    queryClient.resetQueries({
      queryKey: queries.user.current.queryKey,
    });
  }, [queryClient]);

  const {
    isLoaded: isUserUniverseLoaded,
    organizations,
    tenantMemberships,
    isCloudEnabled: userUniverseIsCloudEnabled,
  } = useUserUniverse();

  // Compute current membership and tenant
  const membership = useMemo(() => {
    if (!tenantId || !tenantMemberships) {
      return undefined;
    }

    return tenantMemberships.find((m) => m.tenant?.metadata.id === tenantId);
  }, [tenantId, tenantMemberships]);

  const tenant = membership?.tenant;

  // Update last tenant atom when tenant changes
  useEffect(() => {
    if (tenant && tenant.metadata.id !== lastTenant?.metadata.id) {
      setLastTenant(tenant);
    }
  }, [tenant, lastTenant, setLastTenant]);

  const value = useMemo<AppContextValue>(() => {
    const baseValue = {
      // User
      user: currentUserQuery.data,
      isUserLoaded: currentUserQuery.isSuccess,
      isUserLoading: currentUserQuery.isLoading,
      userError: currentUserQuery.error,
      isUserError: currentUserQuery.isError,
      invalidateCurrentUser,

      // Tenant
      tenant,
      tenantId,
    };

    if (!isUserUniverseLoaded) {
      return {
        ...baseValue,
        isUserUniverseLoaded: false,
        membership: undefined,
        organizations: undefined,
      };
    }

    if (userUniverseIsCloudEnabled) {
      return {
        ...baseValue,
        isUserUniverseLoaded: true,
        isCloudEnabled: true,
        membership: membership?.role,
        organizations: organizations || [],
      };
    }

    return {
      ...baseValue,
      isUserUniverseLoaded: true,
      isCloudEnabled: false,
      membership: membership?.role,
      organizations: undefined,
    };
  }, [
    currentUserQuery.data,
    currentUserQuery.isLoading,
    currentUserQuery.isSuccess,
    currentUserQuery.error,
    currentUserQuery.isError,
    invalidateCurrentUser,
    tenant,
    tenantId,
    isUserUniverseLoaded,
    userUniverseIsCloudEnabled,
    membership?.role,
    organizations,
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
