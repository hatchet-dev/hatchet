import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useOrganizations } from '@/hooks/use-organizations';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  ManagementToken,
  OrganizationInvite,
  OrganizationInviteStatus,
  OrganizationMember,
  OrganizationTenant,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { globalEmitter } from '@/lib/global-emitter';
import { CancelInviteModal } from '@/pages/organizations/$organization/components/cancel-invite-modal';
import { CreateTokenModal } from '@/pages/organizations/$organization/components/create-token-modal';
import { DeleteMemberModal } from '@/pages/organizations/$organization/components/delete-member-modal';
import { DeleteTenantModal } from '@/pages/organizations/$organization/components/delete-tenant-modal';
import { DeleteTokenModal } from '@/pages/organizations/$organization/components/delete-token-modal';
import { appRoutes } from '@/router';
import {
  ArrowRightIcon,
  EllipsisVerticalIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { formatDistanceToNow } from 'date-fns';
import { useEffect, useState } from 'react';

export default function OrganizationSettings() {
  const { tenantId } = useCurrentTenantId();
  const {
    getOrganizationForTenant,
    handleUpdateOrganization,
    updateOrganizationLoading,
  } = useOrganizations();

  const org = getOrganizationForTenant(tenantId);
  const orgId = org?.metadata.id;

  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();

  const [memberToDelete, setMemberToDelete] =
    useState<OrganizationMember | null>(null);
  const [showCreateTokenModal, setShowCreateTokenModal] = useState(false);
  const [tokenToDelete, setTokenToDelete] = useState<ManagementToken | null>(
    null,
  );
  const [inviteToCancel, setInviteToCancel] =
    useState<OrganizationInvite | null>(null);
  const [tenantToArchive, setTenantToArchive] =
    useState<OrganizationTenant | null>(null);
  const [editedName, setEditedName] = useState('');

  const organizationQuery = useQuery({
    ...orgApi.organizationGetQuery(orgId!),
    enabled: !!orgId,
  });

  const managementTokensQuery = useQuery({
    ...orgApi.managementTokenListQuery(orgId!),
    enabled: !!orgId,
  });

  const organizationInvitesQuery = useQuery({
    ...orgApi.organizationInviteListQuery(orgId!),
    enabled: !!orgId,
  });

  const { currentUser } = useCurrentUser();

  const organization = organizationQuery.data;

  useEffect(() => {
    if (organization?.name && !editedName) {
      setEditedName(organization.name);
    }
  }, [organization?.name, editedName]);

  const handleSaveName = () => {
    if (!orgId || !editedName.trim()) {
      return;
    }
    handleUpdateOrganization(orgId, editedName.trim(), () => {
      queryClient.invalidateQueries({ queryKey: ['organization:get', orgId] });
    });
  };

  const formatExpiry = (expiresAt?: string) => {
    if (!expiresAt) {
      return 'never';
    }
    try {
      const expires = new Date(expiresAt);
      return expires < new Date()
        ? 'expired'
        : `in ${formatDistanceToNow(expires)}`;
    } catch {
      return new Date(expiresAt).toLocaleDateString();
    }
  };

  const pendingInvites = organizationInvitesQuery.data?.rows?.filter(
    (i) =>
      i.status === OrganizationInviteStatus.PENDING ||
      i.status === OrganizationInviteStatus.EXPIRED,
  );

  const tenantColumns = [
    {
      columnLabel: 'Name',
      cellRenderer: (
        row: OrganizationTenant & { metadata: { id: string } },
      ) => <span className="font-medium">{row.name || row.id}</span>,
    },
    {
      columnLabel: 'ID',
      cellRenderer: (
        row: OrganizationTenant & { metadata: { id: string } },
      ) => (
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm">{row.id}</span>
          <CopyToClipboard text={row.id} />
        </div>
      ),
    },
    {
      columnLabel: 'Slug',
      cellRenderer: (
        row: OrganizationTenant & { metadata: { id: string } },
      ) => <span className="text-muted-foreground">{row.slug || '-'}</span>,
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (
        row: OrganizationTenant & { metadata: { id: string } },
      ) => <TenantActions row={row} onArchive={setTenantToArchive} />,
    },
  ];

  const memberColumns = [
    {
      columnLabel: 'Email',
      cellRenderer: (row: OrganizationMember) => (
        <span className="font-mono text-sm">{row.email}</span>
      ),
    },
    {
      columnLabel: 'Role',
      cellRenderer: (row: OrganizationMember) => (
        <Badge variant="outline">{row.role}</Badge>
      ),
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (row: OrganizationMember) => (
        <MemberActions
          row={row}
          currentUserEmail={currentUser?.email}
          onDelete={setMemberToDelete}
        />
      ),
    },
  ];

  const inviteColumns = [
    {
      columnLabel: 'Email',
      cellRenderer: (row: OrganizationInvite) => (
        <span className="font-mono text-sm">{row.inviteeEmail}</span>
      ),
    },
    {
      columnLabel: 'Role',
      cellRenderer: (row: OrganizationInvite) => (
        <Badge variant="outline">{row.role}</Badge>
      ),
    },
    {
      columnLabel: 'Status',
      cellRenderer: (row: OrganizationInvite) => (
        <Badge
          variant={
            row.status === OrganizationInviteStatus.PENDING
              ? 'secondary'
              : 'destructive'
          }
        >
          {row.status}
        </Badge>
      ),
    },
    {
      columnLabel: 'Expiry',
      cellRenderer: (row: OrganizationInvite) => (
        <span>{formatExpiry(row.expires)}</span>
      ),
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (row: OrganizationInvite) =>
        row.status === OrganizationInviteStatus.PENDING ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                <EllipsisVerticalIcon className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setInviteToCancel(row)}>
                <TrashIcon className="mr-2 size-4" />
                Cancel Invitation
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null,
    },
  ];

  const tokenColumns = [
    {
      columnLabel: 'Name',
      cellRenderer: (row: ManagementToken & { metadata: { id: string } }) => (
        <span className="font-medium">{row.name}</span>
      ),
    },
    {
      columnLabel: 'Expiry',
      cellRenderer: (row: ManagementToken & { metadata: { id: string } }) => (
        <span>{formatExpiry(row.expiresAt)}</span>
      ),
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (row: ManagementToken & { metadata: { id: string } }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
              <EllipsisVerticalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() =>
                setTokenToDelete({
                  id: row.id,
                  name: row.name,
                  expiresAt: row.expiresAt,
                })
              }
            >
              <TrashIcon className="mr-2 size-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  if (!orgId || !organization) {
    return (
      <div className="h-full w-full flex-grow">
        <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
          <p className="text-sm text-muted-foreground">
            {organizationQuery.isLoading
              ? 'Loading...'
              : 'No organization found for this tenant.'}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex items-center justify-end pb-4">
          <div className="flex items-center gap-2">
            <Input
              value={editedName}
              onChange={(e) => setEditedName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  handleSaveName();
                }
              }}
              className="w-[220px]"
              disabled={updateOrganizationLoading}
            />
            <Button
              size="sm"
              onClick={handleSaveName}
              disabled={updateOrganizationLoading || !editedName.trim()}
            >
              {updateOrganizationLoading && <Spinner />}
              Save
            </Button>
          </div>
        </div>

        <Tabs defaultValue="tenants" className="mt-2">
          <TabsList layout="underlined" className="mb-6">
            <TabsTrigger value="tenants" variant="underlined">
              Tenants
            </TabsTrigger>
            <TabsTrigger value="members" variant="underlined">
              Members
            </TabsTrigger>
            <TabsTrigger value="tokens" variant="underlined">
              Management Tokens
            </TabsTrigger>
          </TabsList>

          <TabsContent value="tenants">
            <div className="flex justify-end mb-4">
              <Button
                size="sm"
                onClick={() =>
                  globalEmitter.emit('create-new-tenant', {
                    defaultOrganizationId: orgId,
                  })
                }
              >
                Add Tenant
              </Button>
            </div>
            {organization.tenants &&
            organization.tenants.filter(
              (t) => t.status !== TenantStatusType.ARCHIVED,
            ).length > 0 ? (
              <SimpleTable
                data={organization.tenants
                  .filter((t) => t.status !== TenantStatusType.ARCHIVED)
                  .map((t) => ({ ...t, metadata: { id: t.id } }))}
                columns={tenantColumns}
              />
            ) : (
              <div className="py-8 text-center text-sm text-muted-foreground">
                No tenants found.
              </div>
            )}
          </TabsContent>

          <TabsContent value="members">
            <div className="space-y-6">
              <div>
                <div className="flex justify-end mb-4">
                  <Button
                    size="sm"
                    onClick={() =>
                      globalEmitter.emit('create-organization-invite', {
                        organizationId: orgId,
                        organizationName: organization.name,
                      })
                    }
                  >
                    Invite Member
                  </Button>
                </div>
                {organization.members && organization.members.length > 0 ? (
                  <SimpleTable
                    data={organization.members}
                    columns={memberColumns}
                  />
                ) : (
                  <div className="py-8 text-center text-sm text-muted-foreground">
                    No members found.
                  </div>
                )}
              </div>

              {pendingInvites && pendingInvites.length > 0 && (
                <div>
                  <SimpleTable data={pendingInvites} columns={inviteColumns} />
                </div>
              )}
            </div>
          </TabsContent>

          <TabsContent value="tokens">
            <div className="flex justify-end mb-4">
              <Button size="sm" onClick={() => setShowCreateTokenModal(true)}>
                Create Token
              </Button>
            </div>
            {managementTokensQuery.data?.rows &&
            managementTokensQuery.data.rows.length > 0 ? (
              <SimpleTable
                data={managementTokensQuery.data.rows.map((t) => ({
                  ...t,
                  metadata: { id: t.id },
                }))}
                columns={tokenColumns}
              />
            ) : (
              <div className="py-8 text-center text-sm text-muted-foreground">
                No management tokens found.
              </div>
            )}
          </TabsContent>
        </Tabs>
      </div>

      {memberToDelete && (
        <DeleteMemberModal
          open={!!memberToDelete}
          onOpenChange={(open) => !open && setMemberToDelete(null)}
          member={memberToDelete}
          organizationName={organization.name}
          onSuccess={() => organizationQuery.refetch()}
        />
      )}

      {orgId && (
        <CreateTokenModal
          open={showCreateTokenModal}
          onOpenChange={setShowCreateTokenModal}
          organizationId={orgId}
          organizationName={organization.name}
          onSuccess={() => managementTokensQuery.refetch()}
        />
      )}

      {tokenToDelete && (
        <DeleteTokenModal
          open={!!tokenToDelete}
          onOpenChange={(open) => !open && setTokenToDelete(null)}
          token={tokenToDelete}
          organizationName={organization.name}
          onSuccess={() => managementTokensQuery.refetch()}
        />
      )}

      {inviteToCancel && (
        <CancelInviteModal
          open={!!inviteToCancel}
          onOpenChange={(open) => !open && setInviteToCancel(null)}
          invite={inviteToCancel}
          organizationName={organization.name}
          onSuccess={() => organizationInvitesQuery.refetch()}
        />
      )}

      {tenantToArchive &&
        (() => {
          const foundTenant = organization.tenants?.find(
            (t) => t?.id === tenantToArchive.id,
          );
          if (!foundTenant) {
            return null;
          }
          return (
            <DeleteTenantModal
              open={!!tenantToArchive}
              onOpenChange={(open) => !open && setTenantToArchive(null)}
              tenant={tenantToArchive}
              tenantName={foundTenant.name || foundTenant.id}
              organizationName={organization.name}
              onSuccess={() => {
                queryClient.invalidateQueries({
                  queryKey: ['organization:get', orgId],
                });
                queryClient.invalidateQueries({ queryKey: ['tenant:get'] });
              }}
            />
          );
        })()}
    </div>
  );
}

function TenantActions({
  row,
  onArchive,
}: {
  row: OrganizationTenant & { metadata: { id: string } };
  onArchive: (tenant: OrganizationTenant) => void;
}) {
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
          <EllipsisVerticalIcon className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          onClick={() =>
            navigate({
              to: appRoutes.tenantRoute.to,
              params: { tenant: row.id },
            })
          }
        >
          <ArrowRightIcon className="mr-2 size-4" />
          View Tenant
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => onArchive({ id: row.id, status: row.status })}
        >
          <TrashIcon className="mr-2 size-4" />
          Archive Tenant
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function MemberActions({
  row,
  currentUserEmail,
  onDelete,
}: {
  row: OrganizationMember;
  currentUserEmail?: string;
  onDelete: (member: OrganizationMember) => void;
}) {
  const isSelf = currentUserEmail === row.email;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
          <EllipsisVerticalIcon className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {isSelf ? (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <DropdownMenuItem
                  disabled
                  className="cursor-not-allowed text-gray-400"
                >
                  <TrashIcon className="mr-2 size-4" />
                  Remove Member
                </DropdownMenuItem>
              </TooltipTrigger>
              <TooltipContent>
                <p>Cannot remove yourself</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : (
          <DropdownMenuItem onClick={() => onDelete(row)}>
            <TrashIcon className="mr-2 size-4" />
            Remove Member
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
