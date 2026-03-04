import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import api from '@/lib/api';
import type { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { Tenant } from '@/lib/api/generated/data-contracts';
import { getCloudMetadataQuery } from '@/pages/auth/hooks/use-cloud';
import { userUniverseQuery } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { PlusIcon } from '@heroicons/react/24/outline';
import { useLoaderData } from '@tanstack/react-router';
import invariant from 'tiny-invariant';

const makeMapOfTenantIdsToTenant = (tenants: Tenant[]) => {
  const map = new Map<string, Tenant>();
  tenants.forEach((t) => {
    map.set(t.metadata.id, t);
  });
  return map;
};

const mapTenantsToOrganizations = (
  organizations: OrganizationForUser[],
  tenantsById: Map<string, Tenant>,
) =>
  organizations.map((org) => ({
    ...org,
    tenants: org.tenants
      .map((t) => tenantsById.get(t.id))
      .filter((t): t is Tenant => t != null),
  }));

type OrgWithTenants = Omit<OrganizationForUser, 'tenants'> & {
  tenants: Tenant[];
};

export const loader = async (): Promise<
  | {
      isCloudEnabled: true;
      organizationsWithTenants: OrgWithTenants[];
    }
  | {
      isCloudEnabled: false;
      tenants: Tenant[];
    }
> => {
  const { isCloudEnabled } = await queryClient.fetchQuery(
    getCloudMetadataQuery,
  );

  const { organizations, tenantMemberships } = await queryClient.fetchQuery(
    userUniverseQuery({ isCloudEnabled, isCloudLoaded: true }),
  );

  const tenants = tenantMemberships
    .map((m) => m.tenant)
    .filter((t): t is Tenant => !!t);

  if (isCloudEnabled) {
    invariant(organizations);
    return {
      isCloudEnabled: true,
      organizationsWithTenants: mapTenantsToOrganizations(
        organizations,
        makeMapOfTenantIdsToTenant(tenants),
      ),
    };
  }

  return {
    isCloudEnabled: false,
    tenants,
  };
};

const tenantColumns = [
  {
    columnLabel: 'Name',
    cellRenderer: (tenant: Tenant) => (
      <span className="font-medium">{tenant.name}</span>
    ),
  },
  {
    columnLabel: 'ID',
    cellRenderer: (tenant: Tenant) => (
      <div className="flex items-center gap-2">
        <span className="font-mono text-sm">{tenant.metadata.id}</span>
        <CopyToClipboard text={tenant.metadata.id} />
      </div>
    ),
  },
  {
    columnLabel: 'Slug',
    cellRenderer: (tenant: Tenant) => (
      <span className="text-muted-foreground">{tenant.slug}</span>
    ),
  },
  {
    columnLabel: 'Actions',
    cellRenderer: () => (
      <Button variant="ghost" size="sm" disabled className="h-8 w-8 p-0">
        ...
      </Button>
    ),
  },
];

export default function OrganizationsPage() {
  const loaderData = useLoaderData({
    from: '/tenants/$tenant/organizations',
  }) as Awaited<ReturnType<typeof loader>>;

  if (!loaderData.isCloudEnabled) {
    const { tenants } = loaderData;

    if (tenants.length === 0) {
      return (
        <div className="py-16 text-center">
          <h3 className="mb-2 text-lg font-medium">No Tenants</h3>
          <p className="text-muted-foreground">
            You are not a member of any tenants.
          </p>
        </div>
      );
    }

    return (
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Tenants</h2>
        <SimpleTable data={tenants} columns={tenantColumns} />
      </div>
    );
  }

  const { organizationsWithTenants } = loaderData;

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
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled>
                Invite to organization...
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled
                leftIcon={<PlusIcon className="size-4" />}
              >
                Add Tenant
              </Button>
            </div>
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
}
