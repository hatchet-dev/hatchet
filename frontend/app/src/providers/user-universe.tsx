import api, { cloudApi } from '@/lib/api/api';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { TenantMember } from '@/lib/api/generated/data-contracts';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createContext, useCallback, useContext, useMemo } from 'react';
import invariant from 'tiny-invariant';

// The user's universe: the tenants they belong to, and if we're in the cloud environment, the organizations those tenants belong to

type UserUniverse = {
  isCloudEnabled: boolean;
  isLoaded: boolean;
  organizations: OrganizationForUserList['rows'] | null;
  tenantMemberships: TenantMember[] | null;
  invalidate: () => void;
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
}: {
  isCloudEnabled: boolean;
  isCloudLoaded: boolean;
}) => ({
  queryKey: ['user-universe', isCloudEnabled],
  queryFn: async (): Promise<PossibleQueryResponses> => {
    const [organizations, tenantMemberships] = await Promise.all([
      isCloudEnabled ? cloudApi.organizationList() : null,
      api.tenantMembershipsList(),
    ]);

    return isCloudEnabled
      ? {
          isCloudEnabled,
          organizations: organizations?.data.rows || [],
          tenantMemberships: tenantMemberships.data.rows || [],
        }
      : {
          isCloudEnabled,
          organizations: null,
          tenantMemberships: tenantMemberships.data.rows || [],
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
  const tenantMembershipAndOrganizationsQuery = useQuery(
    userUniverseQuery({ isCloudEnabled, isCloudLoaded }),
  );

  const queryClient = useQueryClient();

  const invalidate = useCallback(() => {
    queryClient.resetQueries({
      queryKey: ['user-universe'],
    });
  }, [queryClient]);

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
          organizations:
            tenantMembershipAndOrganizationsQuery.data.organizations,
          tenantMemberships:
            tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
          get: getWithOrganizations,
          invalidate,
        };
      }

      return {
        isCloudEnabled,
        isLoaded: tenantMembershipAndOrganizationsAreLoaded,
        organizations: null,
        tenantMemberships: null,
        get: getWithOrganizations,
        invalidate,
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
            organizations: null,
            tenantMemberships:
              tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
            get: getWithoutOrganizations,
            invalidate,
          }
        : {
            isCloudEnabled,
            isLoaded: tenantMembershipAndOrganizationsAreLoaded,
            organizations: null,
            tenantMemberships: null,
            get: getWithoutOrganizations,
            invalidate,
          };
    }
  }, [tenantMembershipAndOrganizationsQuery, isCloudEnabled, get, invalidate]);

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
