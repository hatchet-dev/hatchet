import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import api, { cloudApi, controlPlaneApi } from '@/lib/api/api';
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

// The user's universe: the tenants they belong to, and if we're in the cloud environment, the organizations those tenants belong to

type UserUniverse = {
  isCloudEnabled: boolean;
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
} & (
  | ({
      isCloudEnabled: true;
      get: () => Promise<{
        organizations: OrganizationForUserList['rows'];
        tenantMemberships: TenantMember[];
      }>;
    } & (
      | {
          isLoaded: true;
          organizations: OrganizationForUserList['rows'];
          tenantMemberships: TenantMember[];
        }
      | {
          isLoaded: false;
          organizations: null;
          tenantMemberships: null;
        }
    ))
  | ({
      isCloudEnabled: false;
      organizations: null;
      get: () => Promise<{
        organizations: null;
        tenantMemberships: TenantMember[];
      }>;
    } & (
      | {
          isLoaded: true;
          tenantMemberships: TenantMember[];
        }
      | {
          isLoaded: false;
          tenantMemberships: null;
        }
    ))
);

const UserUniverseContext = createContext<UserUniverse | null>(null);

type PossibleQueryResponses =
  | {
      isCloudEnabled: true;
      organizations: OrganizationForUserList['rows'];
      tenantMemberships: TenantMember[];
    }
  | {
      isCloudEnabled: false;
      organizations: null;
      tenantMemberships: TenantMember[];
    };

export const userUniverseQuery = ({
  isCloudEnabled,
  isCloudLoaded,
  isControlPlaneEnabled,
}: {
  isCloudEnabled: boolean;
  isCloudLoaded: boolean;
  isControlPlaneEnabled: boolean;
}) => ({
  queryKey: ['user-universe', isCloudEnabled, isControlPlaneEnabled],
  queryFn: async (): Promise<PossibleQueryResponses> => {
    const [organizationsResult, tenantMemberships] = await Promise.all([
      isCloudEnabled
        ? isControlPlaneEnabled
          ? controlPlaneApi.organizationList()
          : cloudApi.organizationList()
        : null,
      isControlPlaneEnabled
        ? controlPlaneApi.tenantMembershipsList()
        : api.tenantMembershipsList(),
    ]);

    const organizations = (organizationsResult?.data.rows || []).map((org) => ({
      ...org,
      tenants: org.tenants || [],
    }));
    const membershipRows = tenantMemberships.data.rows || [];

    return isCloudEnabled
      ? {
          isCloudEnabled,
          organizations,
          tenantMemberships: membershipRows,
        }
      : {
          isCloudEnabled,
          organizations: null,
          tenantMemberships: membershipRows,
        };
  },
  enabled: isCloudLoaded,
});

export function UserUniverseProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isCloudEnabled, isCloudLoaded } = useCloud();
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});
  const { isControlPlaneEnabled } = useControlPlane();
  const tenantMembershipAndOrganizationsQuery = useQuery(
    userUniverseQuery({ isCloudEnabled, isCloudLoaded, isControlPlaneEnabled }),
  );

  const queryClient = useQueryClient();

  const invalidate = useCallback(
    () =>
      queryClient.resetQueries({
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
            return result.data;
          }

          throw result.error;
        }),
    [tenantMembershipAndOrganizationsQuery],
  );

  const value = useMemo<UserUniverse>(() => {
    const tenantMembershipAndOrganizationsAreLoaded =
      tenantMembershipAndOrganizationsQuery.isSuccess;
    const isFetching = tenantMembershipAndOrganizationsQuery.isFetching;
    if (isCloudEnabled) {
      const getWithOrganizations = get as () => Promise<{
        organizations: OrganizationForUserList['rows'];
        tenantMemberships: TenantMember[];
      }>;

      if (tenantMembershipAndOrganizationsAreLoaded) {
        invariant(tenantMembershipAndOrganizationsQuery.data.organizations);

        return {
          isCloudEnabled,
          isLoaded: tenantMembershipAndOrganizationsAreLoaded,
          isFetching,
          organizations:
            tenantMembershipAndOrganizationsQuery.data.organizations,
          tenantMemberships:
            tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
          get: getWithOrganizations,
          invalidate,
          logoutMutation,
        };
      }

      return {
        isCloudEnabled,
        isLoaded: tenantMembershipAndOrganizationsAreLoaded,
        isFetching,
        organizations: null,
        tenantMemberships: null,
        get: getWithOrganizations,
        invalidate,
        logoutMutation,
      };
    } else {
      const getWithoutOrganizations = get as () => Promise<{
        organizations: null;
        tenantMemberships: TenantMember[];
      }>;
      return tenantMembershipAndOrganizationsAreLoaded
        ? {
            isCloudEnabled,
            isLoaded: tenantMembershipAndOrganizationsAreLoaded,
            isFetching,
            organizations: null,
            tenantMemberships:
              tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
            get: getWithoutOrganizations,
            invalidate,
            logoutMutation,
          }
        : {
            isCloudEnabled,
            isLoaded: tenantMembershipAndOrganizationsAreLoaded,
            isFetching,
            organizations: null,
            tenantMemberships: null,
            get: getWithoutOrganizations,
            invalidate,
            logoutMutation,
          };
    }
  }, [
    tenantMembershipAndOrganizationsQuery,
    isCloudEnabled,
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
