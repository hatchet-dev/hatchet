import { formatInviteExpiry } from './format-invite-expiry';
import sortByExpires from './sort-by-expires';
import { TenantList, TenantTable } from './tenant-list';
import { Button } from '@/components/v1/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { getCloudMetadataQuery } from '@/hooks/use-cloud';
import api from '@/lib/api';
import { cloudApi, controlPlaneApi, fetchControlPlaneStatus } from '@/lib/api/api';
import type {
  OrganizationForUser,
  OrganizationInvite,
  OrganizationMember,
} from '@/lib/api/generated/cloud/data-contracts';
import { OrganizationInviteStatus } from '@/lib/api/generated/cloud/data-contracts';
import type { CreateOrganizationInviteRequest } from '@/lib/api/generated/cloud/data-contracts';
import {
  Tenant,
  TenantInvite,
  TenantMember,
  TenantMemberRole,
} from '@/lib/api/generated/data-contracts';
import { globalEmitter } from '@/lib/global-emitter';
import { capitalize } from '@/lib/utils';
import { userUniverseQuery } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { PlusIcon } from '@heroicons/react/24/outline';
import { useLoaderData } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
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
      organizationInvites: Map<string, OrganizationInvite[]>;
      tenantMembers: Map<string, null | TenantMember[]>;
      tenantInvites: Map<string, null | TenantInvite[]>;
    }
  | {
      isCloudEnabled: false;
      tenants: TenantWithRole[];
      tenantIdToTenant: Map<string, TenantWithRole>;
      tenantMembers: Map<string, null | TenantMember[]>;
      tenantInvites: Map<string, null | TenantInvite[]>;
    }
> => {
  const { isCloudEnabled } = await queryClient.fetchQuery(
    getCloudMetadataQuery,
  );

  const { isControlPlaneEnabled } = await fetchControlPlaneStatus();

  const { organizations, tenantMemberships } = await queryClient.fetchQuery(
    userUniverseQuery({ isCloudEnabled, isCloudLoaded: true, isControlPlaneEnabled }),
  );

  const tenantIdToTenant = makeMapOfTenantIdsToTenantMember(tenantMemberships);

  const tenantMembers = new Map<string, null | TenantMember[]>();
  const tenantInvites = new Map<string, null | TenantInvite[]>();
  const tenantDataPromise = Promise.all(
    Array.from(tenantIdToTenant.values()).map(async (tenant) => {
      const canManage =
        tenant.currentUsersRole === TenantMemberRole.OWNER ||
        tenant.currentUsersRole === TenantMemberRole.ADMIN;

      if (canManage) {
        const [membersRes, invitesRes] = await Promise.all([
          isControlPlaneEnabled
            ? controlPlaneApi.tenantMemberList(tenant.metadata.id)
            : api.tenantMemberList(tenant.metadata.id),
          isControlPlaneEnabled
            ? controlPlaneApi.tenantInviteList(tenant.metadata.id)
            : api.tenantInviteList(tenant.metadata.id),
        ]);
        tenantMembers.set(tenant.metadata.id, membersRes.data.rows ?? []);
        tenantInvites.set(tenant.metadata.id, invitesRes.data.rows ?? []);
      } else {
        tenantMembers.set(tenant.metadata.id, null);
        tenantInvites.set(tenant.metadata.id, null);
      }
    }),
  );

  if (isCloudEnabled) {
    invariant(organizations);

    const organizationMembers = new Map<string, OrganizationMember[]>();
    const organizationInvites = new Map<string, OrganizationInvite[]>();
    const organizationDataPromise = Promise.all(
      organizations.map(async (org) => {
        const [orgRes, invitesRes] = await Promise.all([
          isControlPlaneEnabled
            ? controlPlaneApi.organizationGet(org.metadata.id)
            : cloudApi.organizationGet(org.metadata.id),
          isControlPlaneEnabled
            ? controlPlaneApi.organizationInviteList(org.metadata.id)
            : cloudApi.organizationInviteList(org.metadata.id),
        ]);
        organizationMembers.set(org.metadata.id, orgRes.data.members ?? []);
        organizationInvites.set(
          org.metadata.id,
          sortByExpires(invitesRes.data.rows ?? []),
        );
      }),
    );

    await Promise.all([tenantDataPromise, organizationDataPromise]);

    return {
      isCloudEnabled: true,
      organizationsWithTenants: mapTenantsToOrganizations(
        organizations,
        tenantIdToTenant,
      ),
      tenantIdToTenant,
      organizationMembers,
      organizationInvites,
      tenantMembers,
      tenantInvites,
    };
  }

  await tenantDataPromise;

  return {
    isCloudEnabled: false,
    tenants: Array.from(tenantIdToTenant.values()),
    tenantIdToTenant,
    tenantMembers,
    tenantInvites,
  };
};

const OrganizationMembersTable = ({
  members,
  invites,
}: {
  members: OrganizationMember[];
  invites: (OrganizationInvite | CreateOrganizationInviteRequest)[];
}) => {
  const pendingInvites = invites.filter(
    (i) => !('status' in i) || i.status === OrganizationInviteStatus.PENDING,
  );

  return (
    <div className="overflow-hidden rounded-md border bg-background">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {members.map((member) => (
            <TableRow key={member.metadata.id}>
              <TableCell>{member.email}</TableCell>
              <TableCell>
                <span className="font-medium">{capitalize(member.role)}</span>
              </TableCell>
            </TableRow>
          ))}
          {pendingInvites.map((invite) => (
            <TableRow
              key={
                'metadata' in invite ? invite.metadata.id : invite.inviteeEmail
              }
              className="text-muted-foreground"
            >
              <TableCell>{invite.inviteeEmail}</TableCell>
              <TableCell>
                Invited{' '}
                {'expires' in invite && formatInviteExpiry(invite.expires)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
};

const OrganizationList = ({
  organizationsWithTenants,
  organizationMembers,
  organizationInvites,
  tenantMembers,
  tenantInvites,
}: {
  organizationsWithTenants: OrgWithTenants[];
  organizationMembers: Map<string, OrganizationMember[]>;
  organizationInvites: Map<
    string,
    (OrganizationInvite | CreateOrganizationInviteRequest)[]
  >;
  tenantMembers: Map<string, null | TenantMember[]>;
  tenantInvites: Map<string, null | TenantInvite[]>;
}) => {
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
    <div className="space-y-12">
      {organizationsWithTenants.map((org) => {
        const members = organizationMembers.get(org.metadata.id) ?? [];
        const invites = organizationInvites.get(org.metadata.id) ?? [];

        return (
          <div key={org.metadata.id} className="space-y-6">
            <h1 className="text-2xl font-bold">{org.name}</h1>

            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">Tenants</h2>
                {org.isOwner && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      globalEmitter.emit('create-new-tenant', {
                        defaultOrganizationId: org.metadata.id,
                      });
                    }}
                    leftIcon={<PlusIcon className="size-4" />}
                  >
                    Add tenant to {org.name}
                  </Button>
                )}
              </div>
              {org.tenants.length > 0 ? (
                <TenantTable
                  tenants={org.tenants}
                  tenantMembers={tenantMembers}
                  tenantInvites={tenantInvites}
                  onInviteMember={(tenantId) =>
                    globalEmitter.emit('create-tenant-invite', { tenantId })
                  }
                />
              ) : (
                <p className="py-4 text-center text-muted-foreground">
                  No tenants in this organization.
                </p>
              )}
            </div>

            {org.isOwner && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-semibold">
                    Organization Members
                  </h2>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      globalEmitter.emit('create-organization-invite', {
                        organizationId: org.metadata.id,
                        organizationName: org.name,
                      })
                    }
                  >
                    Invite new member to {org.name}
                  </Button>
                </div>
                {members.length > 0 || invites.length > 0 ? (
                  <OrganizationMembersTable
                    members={members}
                    invites={invites}
                  />
                ) : (
                  <p className="py-4 text-center text-muted-foreground">
                    No members in this organization.
                  </p>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

const fetchOrganizationInvites = (
  organizationId: string,
): { promise: Promise<OrganizationInvite[]>; cancel: () => void } => {
  const controller = new AbortController();
  const promise = cloudApi
    .organizationInviteList(organizationId, { signal: controller.signal })
    .then((res) => sortByExpires(res.data.rows ?? []));
  return {
    promise,
    cancel: () => controller.abort(),
  };
};

export default function OrganizationsPage() {
  const loaderData = useLoaderData({
    from: '/tenants/$tenant/organizations-and-tenants',
  });

  const [, setOrganizationRefetchPromises] = useState<
    Map<string, ReturnType<typeof fetchOrganizationInvites>>
  >(new Map());

  const [organizationInvites, setOrganizationInvites] = useState<
    Map<string, (OrganizationInvite | CreateOrganizationInviteRequest)[]>
  >(loaderData.isCloudEnabled ? loaderData.organizationInvites : new Map());
  const [tenantInvites, setTenantInvites] = useState(loaderData.tenantInvites);

  useEffect(
    () =>
      globalEmitter.on(
        'organization-invite-created',
        ({ organizationId, invite }) => {
          setOrganizationInvites((prev) => {
            const next = new Map(prev);
            const existing = next.get(organizationId) ?? [];
            next.set(organizationId, [...existing, invite]);
            return next;
          });

          setOrganizationRefetchPromises((previousOrganizationRefetches) => {
            const nextOrganizationRefetches = new Map(
              previousOrganizationRefetches,
            );
            const existingRequest =
              nextOrganizationRefetches.get(organizationId);
            if (existingRequest) {
              existingRequest.cancel();
            }
            const request = fetchOrganizationInvites(organizationId);
            request.promise.then((organizationInvites) =>
              setOrganizationInvites((previousOrganizationInvites) => {
                const nextOrganizationInvites = new Map(
                  previousOrganizationInvites,
                );
                nextOrganizationInvites.set(
                  organizationId,
                  organizationInvites,
                );
                return nextOrganizationInvites;
              }),
            );

            nextOrganizationRefetches.set(organizationId, request);
            return nextOrganizationRefetches;
          });
        },
      ),
    [],
  );

  useEffect(
    () =>
      globalEmitter.on('tenant-invite-created', ({ tenantId, invite }) => {
        setTenantInvites((prev) => {
          const next = new Map(prev);
          const existing = next.get(tenantId) ?? [];
          next.set(tenantId, [...existing, invite]);
          return next;
        });
      }),
    [],
  );

  if (!loaderData.isCloudEnabled) {
    return (
      <TenantList
        tenants={loaderData.tenants}
        tenantMembers={loaderData.tenantMembers}
        tenantInvites={tenantInvites}
      />
    );
  }

  return (
    <OrganizationList
      organizationsWithTenants={loaderData.organizationsWithTenants}
      organizationMembers={loaderData.organizationMembers}
      organizationInvites={organizationInvites}
      tenantMembers={loaderData.tenantMembers}
      tenantInvites={tenantInvites}
    />
  );
}
