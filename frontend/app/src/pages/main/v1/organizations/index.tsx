import { makeTenantColumns, TenantList } from './tenant-list';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import type {
  OrganizationForUser,
  OrganizationMember,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  Tenant,
  TenantMember,
  TenantMemberRole,
} from '@/lib/api/generated/data-contracts';
import { globalEmitter } from '@/lib/global-emitter';
import { getCloudMetadataQuery } from '@/pages/auth/hooks/use-cloud';
import { userUniverseQuery } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { PlusIcon } from '@heroicons/react/24/outline';
import { useLoaderData, useNavigate } from '@tanstack/react-router';
import invariant from 'tiny-invariant';

export type TenantWithRole = Tenant & {
  currentUsersRole: TenantMemberRole;
};

const makeMapOfTenantIdsToTenantMember = (tenants: TenantMember[]) => {
  const map = new Map<string, TenantWithRole>();
  tenants.forEach((t) => {
    if (t.tenant) {
      map.set(t.tenant.metadata.id, {
        ...t.tenant,
        currentUsersRole: t.role,
      });
    }
  });
  return map;
};

const mapTenantsToOrganizations = (
  organizations: OrganizationForUser[],
  tenantsById: Map<string, TenantWithRole>,
) =>
  organizations.map((org) => ({
    ...org,
    tenants: org.tenants
      .map((t) => tenantsById.get(t.id))
      .filter((t): t is TenantWithRole => t != null),
  }));

type OrgWithTenants = Omit<OrganizationForUser, 'tenants'> & {
  tenants: TenantWithRole[];
};

export const loader = async (): Promise<
  | {
      isCloudEnabled: true;
      organizationsWithTenants: OrgWithTenants[];
      tenantIdToTenant: Map<string, TenantWithRole>;
      organizationMembers: Map<string, OrganizationMember[]>;
      tenantMembers: Map<string, null | TenantMember[]>;
    }
  | {
      isCloudEnabled: false;
      tenants: TenantWithRole[];
      tenantIdToTenant: Map<string, TenantWithRole>;
      tenantMembers: Map<string, null | TenantMember[]>;
    }
> => {
  const { isCloudEnabled } = await queryClient.fetchQuery(
    getCloudMetadataQuery,
  );

  const { organizations, tenantMemberships } = await queryClient.fetchQuery(
    userUniverseQuery({ isCloudEnabled, isCloudLoaded: true }),
  );

  const tenantIdToTenant = makeMapOfTenantIdsToTenantMember(tenantMemberships);

  const tenantMembers = new Map<string, null | TenantMember[]>();
  const tenantMembersPromise = Promise.all(
    Array.from(tenantIdToTenant.values()).map(async (tenant) => {
      const memberList =
        tenant.currentUsersRole === TenantMemberRole.OWNER ||
        tenant.currentUsersRole === TenantMemberRole.ADMIN
          ? ((await api.tenantMemberList(tenant.metadata.id)).data.rows ?? [])
          : null;
      tenantMembers.set(tenant.metadata.id, memberList);
    }),
  ).then(() => tenantMembers);

  if (isCloudEnabled) {
    invariant(organizations);

    const organizationMembers = new Map<string, OrganizationMember[]>();
    const organizationMembersPromise = Promise.all(
      organizations.map(async (org) => {
        const res = await cloudApi.organizationGet(org.metadata.id);
        organizationMembers.set(org.metadata.id, res.data.members ?? []);
      }),
    ).then(() => organizationMembers);

    return {
      isCloudEnabled: true,
      organizationsWithTenants: mapTenantsToOrganizations(
        organizations,
        tenantIdToTenant,
      ),
      tenantIdToTenant,
      organizationMembers: await organizationMembersPromise,
      tenantMembers: await tenantMembersPromise,
    };
  }

  return {
    isCloudEnabled: false,
    tenants: Array.from(tenantIdToTenant.values()),
    tenantIdToTenant,
    tenantMembers: await tenantMembersPromise,
  };
};

const OrganizationList = ({
  organizationsWithTenants,
}: {
  organizationsWithTenants: OrgWithTenants[];
}) => {
  const navigate = useNavigate();

  const tenantColumns = makeTenantColumns({
    onViewTenant: (tenantId) =>
      navigate({
        to: appRoutes.tenantRoute.to,
        params: { tenant: tenantId },
      }),
    onInviteMember: (tenantId) =>
      globalEmitter.emit('create-tenant-invite', { tenantId }),
  });

  if (organizationsWithTenants.length === 0) {
    return (
      <div className="py-16 text-center">
        <h3 className="mb-2 text-lg font-medium">No Organizations</h3>
        <p className="text-muted-foreground">
          You are not a member of any organizations.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {organizationsWithTenants.map((org) => (
        <div key={org.metadata.id} className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">{org.name}</h2>
            {org.isOwner && (
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled>
                  Invite to organization...
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    globalEmitter.emit('new-tenant', {
                      defaultOrganizationId: org.metadata.id,
                    });
                  }}
                  leftIcon={<PlusIcon className="size-4" />}
                >
                  Add Tenant
                </Button>
              </div>
            )}
          </div>

          {org.tenants.length > 0 ? (
            <SimpleTable data={org.tenants} columns={tenantColumns} />
          ) : (
            <p className="py-4 text-center text-muted-foreground">
              No tenants in this organization.
            </p>
          )}
        </div>
      ))}
    </div>
  );
};

export default function OrganizationsPage() {
  const loaderData = useLoaderData({
    from: '/tenants/$tenant/organizations',
  }) as Awaited<ReturnType<typeof loader>>;

  if (!loaderData.isCloudEnabled) {
    return <TenantList tenants={loaderData.tenants} />;
  }

  return (
    <OrganizationList
      organizationsWithTenants={loaderData.organizationsWithTenants}
    />
  );
}
