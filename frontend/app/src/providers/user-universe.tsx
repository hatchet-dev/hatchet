import api, { cloudApi } from '@/lib/api/api';
import {
  OrganizationForUser,
  OrganizationForUserList,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { Tenant, TenantMember } from '@/lib/api/generated/data-contracts';
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
          getOrganizationForTenant: (tenantId: string) => OrganizationForUser;
          getTenantWithTenantId: (tenantId: string) => Tenant;
        }
      | {
          isLoaded: false;
          organizations: null;
          tenantMemberships: null;
          getOrganizationForTenant: null;
          getTenantWithTenantId: null;
        }
    ))
  | ({
      isCloudEnabled: false;
      organizations: null;
      get: () => Promise<{
        organizations: null;
        tenantMemberships: TenantMember[];
      }>;
      getOrganizationForTenant: null;
    } & (
      | {
          isLoaded: true;
          tenantMemberships: TenantMember[];
          getTenantWithTenantId: (tenantId: string) => Tenant;
        }
      | {
          isLoaded: false;
          tenantMemberships: null;
          getTenantWithTenantId: null;
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
    const [organizationsResponse, tenantMembershipsResponse] =
      await Promise.all([
        isCloudEnabled ? cloudApi.organizationList() : null,
        api.tenantMembershipsList(),
      ]);

    const tenantMembershipRows = tenantMembershipsResponse.data.rows || [];
    const organizations = organizationsResponse?.data.rows || [];

    const archivedTenantIds =
      organizations &&
      new Set(
        organizations.flatMap((organization) =>
          organization.tenants
            .filter((tenant) => tenant.status === TenantStatusType.ARCHIVED)
            .map((tenant) => tenant.id),
        ),
      );
    const tenantMemberships = archivedTenantIds
      ? tenantMembershipRows.filter(
          (membership) =>
            membership.tenant &&
            !archivedTenantIds.has(membership.tenant.metadata.id),
        )
      : tenantMembershipRows;

    return isCloudEnabled
      ? {
          isCloudEnabled,
          organizations,
          tenantMemberships,
        }
      : {
          isCloudEnabled,
          organizations: null,
          tenantMemberships,
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

    let getTenantWithTenantId: ((tenantId: string) => Tenant) | null = null;

    if (tenantMembershipAndOrganizationsAreLoaded) {
      const tenantIdToTenant = new Map<string, Tenant>(
        tenantMembershipAndOrganizationsQuery.data.tenantMemberships.map(
          (membership) => {
            invariant(membership.tenant);
            return [membership.tenant.metadata.id, membership.tenant];
          },
        ),
      );
      getTenantWithTenantId = (tenantId: string) => {
        const tenant = tenantIdToTenant.get(tenantId);
        invariant(tenant);
        return tenant;
      };
    }

    if (isCloudEnabled) {
      const getWithOrganizations = get as () => Promise<{
        organizations: OrganizationForUserList['rows'];
        tenantMemberships: TenantMember[];
      }>;

      if (tenantMembershipAndOrganizationsAreLoaded) {
        const organizations =
          tenantMembershipAndOrganizationsQuery.data.organizations;
        invariant(organizations);

        const tenantIdToOrganization = new Map<string, OrganizationForUser>(
          organizations.flatMap((organization) =>
            organization.tenants.map((tenant) => [tenant.id, organization]),
          ),
        );
        const getOrganizationForTenant = (tenantId: string) => {
          const organization = tenantIdToOrganization.get(tenantId);
          invariant(organization);
          return organization;
        };

        invariant(getTenantWithTenantId);

        return {
          isCloudEnabled,
          isLoaded: tenantMembershipAndOrganizationsAreLoaded,
          organizations: organizations,
          tenantMemberships:
            tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
          get: getWithOrganizations,
          invalidate,
          getOrganizationForTenant,
          getTenantWithTenantId,
        };
      }

      return {
        isCloudEnabled,
        isLoaded: tenantMembershipAndOrganizationsAreLoaded,
        organizations: null,
        tenantMemberships: null,
        get: getWithOrganizations,
        invalidate,
        getOrganizationForTenant: null,
        getTenantWithTenantId: null,
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
            getOrganizationForTenant: null,
            getTenantWithTenantId: getTenantWithTenantId!, // wtf typescript come on
          }
        : {
            isCloudEnabled,
            isLoaded: tenantMembershipAndOrganizationsAreLoaded,
            organizations: null,
            tenantMemberships: null,
            get: getWithoutOrganizations,
            invalidate,
            getOrganizationForTenant: null,
            getTenantWithTenantId: null,
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
