import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/v1/ui/accordion';
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
import { TenantInvite, TenantMember } from '@/lib/api';
import {
  ManagementToken,
  OrganizationInvite,
  OrganizationInviteStatus,
  OrganizationMember,
  OrganizationTenant,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
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
  const [expandedTenantIds, setExpandedTenantIds] = useState<string[]>([]);
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
            <div className="mb-4 flex justify-end">
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
            {organization.tenants?.filter(
              (t) => t.status !== TenantStatusType.ARCHIVED,
            ).length ? (
              <Accordion
                type="multiple"
                value={expandedTenantIds}
                onValueChange={setExpandedTenantIds}
                className="space-y-3 rounded-md border bg-background p-3"
              >
                {organization.tenants
                  .filter((t) => t.status !== TenantStatusType.ARCHIVED)
                  .map((tenant) => (
                    <TenantAccordionItem
                      key={tenant.id}
                      tenant={tenant}
                      isExpanded={expandedTenantIds.includes(tenant.id)}
                      onArchive={setTenantToArchive}
                    />
                  ))}
              </Accordion>
            ) : (
              <div className="py-8 text-center text-sm text-muted-foreground">
                No tenants found.
              </div>
            )}
          </TabsContent>

          <TabsContent value="members">
            <div className="space-y-6">
              <div>
                <div className="mb-4 flex justify-end">
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
            <div className="mb-4 flex justify-end">
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

function TenantAccordionItem({
  tenant,
  isExpanded,
  onArchive,
}: {
  tenant: OrganizationTenant;
  isExpanded: boolean;
  onArchive: (tenant: OrganizationTenant) => void;
}) {
  const { tenantMemberListQuery, tenantInviteListQuery } = useTenantApi();

  const membersQuery = useQuery({
    ...tenantMemberListQuery(tenant.id),
    enabled: isExpanded,
  });

  const invitesQuery = useQuery({
    ...tenantInviteListQuery(tenant.id),
    enabled: isExpanded,
  });

  const tenantMembers = membersQuery.data?.rows || [];
  const tenantInvites = invitesQuery.data?.rows || [];

  return (
    <AccordionItem value={tenant.id} className="overflow-hidden bg-background">
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        <AccordionTrigger className="flex-1 py-1 hover:no-underline [&>svg]:text-muted-foreground">
          <div className="min-w-0 text-left">
            <p className="truncate font-medium leading-5">
              {tenant.name || tenant.id}
            </p>
          </div>
        </AccordionTrigger>

        <div className="flex items-center gap-2">
          <div className="hidden items-center gap-2 lg:flex">
            <span className="font-mono text-xs text-muted-foreground">
              {tenant.id}
            </span>
            <CopyToClipboard text={tenant.id} />
          </div>
          <TenantActions
            row={{ ...tenant, metadata: { id: tenant.id } }}
            onArchive={onArchive}
          />
        </div>
      </div>

      <AccordionContent className="border-border/70 px-4 pb-4 pt-4">
        <div className="space-y-5">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-medium">Members</h4>
            <Button
              size="sm"
              onClick={() =>
                globalEmitter.emit('create-tenant-invite', {
                  tenantId: tenant.id,
                })
              }
            >
              Invite Member
            </Button>
          </div>

          {membersQuery.isLoading ? (
            <div className="py-4 text-sm text-muted-foreground">
              Loading members...
            </div>
          ) : tenantMembers.length > 0 ? (
            <TenantMemberList members={tenantMembers} />
          ) : (
            <div className="py-4 text-sm text-muted-foreground">
              No members found.
            </div>
          )}

          <div className="space-y-2">
            <h4 className="text-sm font-medium">Pending Invites</h4>
            {invitesQuery.isLoading ? (
              <div className="py-2 text-sm text-muted-foreground">
                Loading invites...
              </div>
            ) : tenantInvites.length > 0 ? (
              <TenantInviteList invites={tenantInvites} />
            ) : (
              <div className="py-2 text-sm text-muted-foreground">
                No pending invites.
              </div>
            )}
          </div>
        </div>
      </AccordionContent>
    </AccordionItem>
  );
}

function TenantMemberList({ members }: { members: TenantMember[] }) {
  return (
    <div className="rounded-md border border-border/70">
      <div className="hidden grid-cols-[minmax(0,1.2fr)_minmax(0,1.6fr)_140px_140px] gap-3 border-b border-border/70 bg-muted/20 px-4 py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground md:grid">
        <span>Name</span>
        <span>Email</span>
        <span>Role</span>
        <span>Joined</span>
      </div>
      <div>
        {members.map((member) => (
          <div
            key={member.metadata.id}
            className="grid gap-3 border-b border-border/50 px-4 py-3 last:border-b-0 md:grid-cols-[minmax(0,1.2fr)_minmax(0,1.6fr)_140px_140px] md:items-center"
          >
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Name</p>
              <p className="font-medium">{member.user.name}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Email</p>
              <p className="font-mono text-sm">{member.user.email}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Role</p>
              <Badge variant="outline">{member.role}</Badge>
            </div>
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Joined</p>
              <RelativeDate date={member.metadata.createdAt} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function TenantInviteList({ invites }: { invites: TenantInvite[] }) {
  return (
    <div className="rounded-md border border-border/70">
      <div className="hidden grid-cols-[minmax(0,1.6fr)_120px_140px_140px] gap-3 border-b border-border/70 bg-muted/20 px-4 py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground md:grid">
        <span>Email</span>
        <span>Role</span>
        <span>Created</span>
        <span>Expires</span>
      </div>
      <div>
        {invites.map((invite) => (
          <div
            key={invite.metadata.id}
            className="grid gap-3 border-b border-border/50 px-4 py-3 last:border-b-0 md:grid-cols-[minmax(0,1.6fr)_120px_140px_140px] md:items-center"
          >
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Email</p>
              <p className="font-mono text-sm">{invite.email}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Role</p>
              <Badge variant="outline">{invite.role}</Badge>
            </div>
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Created</p>
              <RelativeDate date={invite.metadata.createdAt} />
            </div>
            <div>
              <p className="text-xs text-muted-foreground md:hidden">Expires</p>
              <RelativeDate date={invite.expires} />
            </div>
          </div>
        ))}
      </div>
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
