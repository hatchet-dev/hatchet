import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import type { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { TenantStatusType } from '@/lib/api/generated/cloud/data-contracts';
import { Tenant } from '@/lib/api/generated/data-contracts';
import { globalEmitter } from '@/lib/global-emitter';
import { getCloudMetadataQuery } from '@/pages/auth/hooks/use-cloud';
import { DeleteTenantModal } from '@/pages/organizations/$organization/components/delete-tenant-modal';
import { userUniverseQuery, useUserUniverse } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import {
  EllipsisVerticalIcon,
  TrashIcon,
  PlusIcon,
} from '@heroicons/react/24/outline';
import { Link, useLoaderData } from '@tanstack/react-router';
import { useState } from 'react';
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
      tenantIdToTenant: Map<string, Tenant>;
    }
  | {
      isCloudEnabled: false;
      tenants: Tenant[];
      tenantIdToTenant: Map<string, Tenant>;
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

  const tenantIdToTenant = makeMapOfTenantIdsToTenant(tenants);

  if (isCloudEnabled) {
    invariant(organizations);
    return {
      isCloudEnabled: true,
      organizationsWithTenants: mapTenantsToOrganizations(
        organizations,
        tenantIdToTenant,
      ),
      tenantIdToTenant,
    };
  }

  return {
    isCloudEnabled: false,
    tenants,
    tenantIdToTenant,
  };
};

const makeTenantColumns = (onArchive?: (tenant: Tenant) => void) => [
  {
    columnLabel: 'Name',
    cellRenderer: (tenant: Tenant) => (
      <Link
        to={appRoutes.tenantRoute.to}
        params={{ tenant: tenant.metadata.id }}
        className="font-medium hover:underline"
      >
        {tenant.name}
      </Link>
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
    cellRenderer: (tenant: Tenant) => (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
            <EllipsisVerticalIcon className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {onArchive && (
            <DropdownMenuItem onClick={() => onArchive(tenant)}>
              <TrashIcon className="mr-2 size-4" />
              Archive Tenant
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  },
];

const TenantList = ({ tenants }: { tenants: Tenant[] }) => {
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
      <SimpleTable data={tenants} columns={makeTenantColumns()} />
    </div>
  );
};

const OrganizationList = ({
  organizationsWithTenants,
}: {
  organizationsWithTenants: OrgWithTenants[];
}) => {
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
  const [tenantToArchive, setTenantToArchive] = useState<{
    tenant: Tenant;
    orgName: string;
  } | null>(null);

  const tenantColumns = makeTenantColumns((tenant) => {
    const org = organizationsWithTenants.find((o) =>
      o.tenants.some((t) => t.metadata.id === tenant.metadata.id),
    );
    setTenantToArchive({ tenant, orgName: org?.name ?? '' });
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
    <>
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

      {tenantToArchive && (
        <DeleteTenantModal
          open={!!tenantToArchive}
          onOpenChange={(open) => !open && setTenantToArchive(null)}
          tenant={{
            id: tenantToArchive.tenant.metadata.id,
            status: TenantStatusType.ACTIVE,
          }}
          tenantName={tenantToArchive.tenant.name}
          organizationName={tenantToArchive.orgName}
          onSuccess={() => {
            invalidateUserUniverse();
            setTenantToArchive(null);
          }}
        />
      )}
    </>
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
