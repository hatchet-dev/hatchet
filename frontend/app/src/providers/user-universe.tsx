import api, { cloudApi } from '@/lib/api/api';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { TenantMember } from '@/lib/api/generated/data-contracts';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createContext, useCallback, useContext, useMemo } from 'react';
import invariant from 'tiny-invariant';

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

export function UserUniverseProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isCloudEnabled } = useCloud();
  const queryClient = useQueryClient();
  const tenantMembershipAndOrganizationsQuery = useQuery({
    queryKey: ['user-universe'],
    queryFn: async () => {
      const [organizations, tenantMemberships] = await Promise.all([
        isCloudEnabled ? cloudApi.organizationList() : null,
        api.tenantMembershipsList(),
      ]);
      return {
        organizations: isCloudEnabled ? organizations?.data.rows || [] : null,
        tenantMemberships: tenantMemberships.data.rows || [],
      };
    },
  });

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['user-universe'],
    });
  }, []);

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
    [],
  );

  const value = useMemo<UserUniverse>(() => {
    const isLoaded = tenantMembershipAndOrganizationsQuery.isSuccess;
    if (isCloudEnabled) {
      const getWithOrganizations = get as () => Promise<{
        organizations: OrganizationForUserList['rows'];
        tenantMemberships: TenantMember[];
      }>;

      return isLoaded
        ? {
            isCloudEnabled,
            isLoaded,
            organizations: tenantMembershipAndOrganizationsQuery.data
              .organizations as OrganizationForUserList['rows'],
            tenantMemberships:
              tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
            get: getWithOrganizations,
            invalidate,
          }
        : {
            isCloudEnabled,
            isLoaded,
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
      return isLoaded
        ? {
            isCloudEnabled,
            isLoaded,
            organizations: null,
            tenantMemberships:
              tenantMembershipAndOrganizationsQuery.data.tenantMemberships,
            get: getWithoutOrganizations,
            invalidate,
          }
        : {
            isCloudEnabled,
            isLoaded,
            organizations: null,
            tenantMemberships: null,
            get: getWithoutOrganizations,
            invalidate,
          };
    }
  }, [
    tenantMembershipAndOrganizationsQuery.data,
    tenantMembershipAndOrganizationsQuery.isSuccess,
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
