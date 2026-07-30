import useControlPlane from '@/hooks/use-control-plane';
import api, { controlPlaneApi } from '@/lib/api/api';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { TenantMember } from '@/lib/api/generated/data-contracts';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { createContext, useCallback, useContext, useMemo } from 'react';
import invariant from 'tiny-invariant';

// The user's universe: the tenants they belong to and, under the control plane,
// the organizations those tenants belong to.
// Control-plane status comes from useControlPlane — not re-exported here.

type UserUniverse = {
  isLoaded: boolean;
  isFetching: boolean;
  organizations: OrganizationForUserList['rows'] | null;
  tenantMemberships: TenantMember[] | null;
  invalidate: () => Promise<void>;
  logoutMutation: UseMutationResult<
    void,
    AxiosError<unknown, any>,
    void,
    unknown
  >;
  get: () => Promise<{
    organizations: OrganizationForUserList['rows'] | null;
    tenantMemberships: TenantMember[];
  }>;
} & (
  | {
      isLoaded: true;
      tenantMemberships: TenantMember[];
      // null when self-hosted; rows (possibly empty) when control plane is on
      organizations: OrganizationForUserList['rows'] | null;
    }
  | {
      isLoaded: false;
      organizations: null;
      tenantMemberships: null;
    }
);

const UserUniverseContext = createContext<UserUniverse | null>(null);

type PossibleQueryResponses =
  | {
      isControlPlaneEnabled: true;
      organizations: OrganizationForUserList['rows'];
      tenantMemberships: TenantMember[];
    }
  | {
      isControlPlaneEnabled: false;
      organizations: null;
      tenantMemberships: TenantMember[];
    };

export const userUniverseQuery = (isControlPlaneEnabled: boolean) => ({
  queryKey: ['user-universe', isControlPlaneEnabled],
  queryFn: async (): Promise<PossibleQueryResponses> => {
    const [organizationsResult, tenantMemberships] = await Promise.all([
      isControlPlaneEnabled ? controlPlaneApi.organizationList() : null,
      isControlPlaneEnabled
        ? controlPlaneApi.tenantMembershipsList()
        : api.tenantMembershipsList(),
    ]);

    const organizations = (organizationsResult?.data.rows || []).map((org) => ({
      ...org,
      tenants: org.tenants || [],
    }));
    const membershipRows = tenantMemberships.data.rows || [];

    return isControlPlaneEnabled
      ? {
          isControlPlaneEnabled,
          organizations,
          tenantMemberships: membershipRows,
        }
      : {
          isControlPlaneEnabled,
          organizations: null,
          tenantMemberships: membershipRows,
        };
  },
  refetchInterval: 30_000,
  staleTime: 30_000,
});

export function UserUniverseProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const tenantMembershipAndOrganizationsQuery = useQuery({
    ...userUniverseQuery(isControlPlaneEnabled),
    enabled: !isControlPlaneLoading,
  });

  const queryClient = useQueryClient();

  // invalidate (not reset): resetting wipes tenantMemberships to null mid-
  // refetch, which churns every effect keyed on it (e.g. the invite modal
  // auto-open in authenticated.tsx) and flickers isLoaded app-wide.
  const invalidate = useCallback(
    () =>
      queryClient.invalidateQueries({
        queryKey: ['user-universe'],
      }),
    [queryClient],
  );

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      if (isControlPlaneEnabled) {
        await controlPlaneApi.cloudUserUpdateLogout();
        return;
      }
      await api.userUpdateLogout();
    },
    onError: handleApiError,
    onSettled: () => {
      // always clear on logout attempt, even if the request fails
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  const get = useCallback(
    () =>
      tenantMembershipAndOrganizationsQuery
        .refetch({
          cancelRefetch: false,
        })
        .then((result) => {
          if (result.isSuccess) {
            return {
              organizations: result.data.organizations,
              tenantMemberships: result.data.tenantMemberships,
            };
          }

          throw result.error;
        }),
    [tenantMembershipAndOrganizationsQuery],
  );

  const value = useMemo<UserUniverse>(() => {
    const isLoaded = tenantMembershipAndOrganizationsQuery.isSuccess;
    const isFetching = tenantMembershipAndOrganizationsQuery.isFetching;

    if (isLoaded) {
      if (isControlPlaneEnabled) {
        invariant(tenantMembershipAndOrganizationsQuery.data.organizations);

        return {
          isLoaded: true,
          isFetching,
          organizations:
            tenantMembershipAndOrganizationsQuery.data.organizations,
          tenantMemberships:
            tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
          get,
          invalidate,
          logoutMutation,
        };
      }

      return {
        isLoaded: true,
        isFetching,
        organizations: null,
        tenantMemberships:
          tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
        get,
        invalidate,
        logoutMutation,
      };
    }

    return {
      isLoaded: false,
      isFetching,
      organizations: null,
      tenantMemberships: null,
      get,
      invalidate,
      logoutMutation,
    };
  }, [
    tenantMembershipAndOrganizationsQuery,
    isControlPlaneEnabled,
    get,
    invalidate,
    logoutMutation,
  ]);

  return (
    <UserUniverseContext.Provider value={value}>
      {children}
    </UserUniverseContext.Provider>
  );
}

export function useUserUniverse() {
  const context = useContext(UserUniverseContext);
  invariant(
    context,
    'useUserUniverse must be used within UserUniverseProvider',
  );
  return context;
}
