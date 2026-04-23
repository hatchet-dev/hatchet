import { SettingsPageHeader } from '../components/settings-page-header';
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
import { Dialog } from '@/components/v1/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
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
import { TenantInvite, TenantMember, TenantMemberRole } from '@/lib/api';
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
import { useApiError } from '@/lib/hooks';
import { MemberActions as TenantMemberActions } from '@/pages/main/v1/tenant-settings/members/components/members-columns';
import { UpdateMemberForm } from '@/pages/main/v1/tenant-settings/members/components/update-member-form';
import { CancelInviteModal } from '@/pages/organizations/$organization/components/cancel-invite-modal';
import { CreateTokenModal } from '@/pages/organizations/$organization/components/create-token-modal';
import { DeleteMemberModal } from '@/pages/organizations/$organization/components/delete-member-modal';
import { DeleteTenantModal } from '@/pages/organizations/$organization/components/delete-tenant-modal';
import { DeleteTokenModal } from '@/pages/organizations/$organization/components/delete-token-modal';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import {
  ArrowRightIcon,
  CheckIcon,
  EllipsisVerticalIcon,
  PencilSquareIcon,
  TrashIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { formatDistanceToNow } from 'date-fns';
import { useMemo, useState } from 'react';

export default function OrganizationSettings() {
  const { isCloudEnabled } = useOrganizations();
  return isCloudEnabled ? (
    <CloudOrganizationSettings />
  ) : (
    <OssOrganizationSettings />
  );
}

function CloudOrganizationSettings() {
  const { tenantId } = useCurrentTenantId();
  const {
    getOrganizationForTenant,
    handleUpdateOrganization,
    updateOrganizationLoading,
  } = useOrganizations();

  const org = getOrganizationForTenant(tenantId);
  const orgId = org?.metadata.id;
  const isOrganizationOwner = org?.isOwner ?? false;

  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { currentUser } = useCurrentUser();

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
  const [isEditingName, setIsEditingName] = useState(false);

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

  const organization = organizationQuery.data;
  const organizationName = organization?.name ?? org?.name ?? '';
  const currentOrganizationName = organization?.name ?? '';

  const visibleTenants = useMemo(
    () =>
      (organization?.tenants ?? org?.tenants)?.filter(
        (tenant) => tenant.status !== TenantStatusType.ARCHIVED,
      ) || [],
    [org?.tenants, organization?.tenants],
  );

  const handleSaveName = () => {
    const trimmedName = editedName.trim();

    if (!orgId || !trimmedName) {
      return;
    }

    if (trimmedName === currentOrganizationName) {
      setIsEditingName(false);
      return;
    }

    handleUpdateOrganization(orgId, trimmedName, () => {
      setEditedName(trimmedName);
      setIsEditingName(false);
      queryClient.invalidateQueries({ queryKey: ['organization:get', orgId] });
    });
  };

  const handleStartEditingName = () => {
    setEditedName(currentOrganizationName);
    setIsEditingName(true);
  };

  const handleCancelEditingName = () => {
    setEditedName(currentOrganizationName);
    setIsEditingName(false);
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
    ...(isOrganizationOwner
      ? [
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
        ]
      : []),
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
    ...(isOrganizationOwner
      ? [
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
        ]
      : []),
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
    ...(isOrganizationOwner
      ? [
          {
            columnLabel: 'Actions',
            cellRenderer: (
              row: ManagementToken & { metadata: { id: string } },
            ) => (
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
        ]
      : []),
  ];

  if (!orgId) {
    return (
      <div className="h-full w-full flex-grow">
        <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
          <p className="text-sm text-muted-foreground">
            No organization found for this tenant.
          </p>
        </div>
      </div>
    );
  }

  if (organizationQuery.isLoading && !organization) {
    return (
      <div className="h-full w-full flex-grow">
        <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Organization settings"
          description={
            isOrganizationOwner
              ? 'Update the organization name and manage tenants, members, and management tokens.'
              : 'Review the tenants associated with this organization.'
          }
        >
          {isOrganizationOwner && (
            <div className="w-full rounded-lg border border-border/50 bg-muted/10 p-4 md:max-w-sm">
              <div className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Organization name
              </div>

              {isEditingName ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={editedName}
                    onChange={(e) => setEditedName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        handleSaveName();
                      }

                      if (e.key === 'Escape') {
                        handleCancelEditingName();
                      }
                    }}
                    className="h-10 flex-1 bg-background/60"
                    disabled={updateOrganizationLoading}
                    aria-label="Organization name"
                    autoFocus
                  />

                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={handleCancelEditingName}
                      disabled={updateOrganizationLoading}
                      hoverText="Cancel editing"
                      className="shrink-0 hover:bg-muted/50"
                    >
                      <XMarkIcon className="size-4" />
                    </Button>
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={handleSaveName}
                      disabled={
                        updateOrganizationLoading ||
                        !editedName.trim() ||
                        editedName.trim() === organizationName
                      }
                      hoverText="Save organization name"
                      className="shrink-0 bg-background/60 hover:bg-muted/50"
                    >
                      {updateOrganizationLoading ? (
                        <Spinner />
                      ) : (
                        <CheckIcon className="size-4" />
                      )}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <div className="flex h-10 min-w-0 flex-1 items-center rounded-md border border-input bg-background/60 px-3">
                    <p className="truncate text-sm font-medium text-foreground">
                      {organizationName}
                    </p>
                  </div>

                  <Button
                    variant="outline"
                    size="icon"
                    onClick={handleStartEditingName}
                    hoverText="Edit organization name"
                    className="shrink-0 bg-background/60 hover:bg-muted/50"
                  >
                    <PencilSquareIcon className="size-4" />
                  </Button>
                </div>
              )}
            </div>
          )}
        </SettingsPageHeader>

        <Tabs defaultValue="tenants" className="mt-2">
          <TabsList layout="underlined" className="mb-6">
            <TabsTrigger value="tenants" variant="underlined">
              Tenants
            </TabsTrigger>
            <TabsTrigger value="members" variant="underlined">
              Organization Members
            </TabsTrigger>
            <TabsTrigger value="tokens" variant="underlined">
              Management Tokens
            </TabsTrigger>
          </TabsList>

          <TabsContent value="tenants">
            <TenantsSection
              tenants={visibleTenants}
              expandedTenantIds={expandedTenantIds}
              setExpandedTenantIds={setExpandedTenantIds}
              onArchive={setTenantToArchive}
              defaultOrganizationId={orgId}
              canManageOrganization={isOrganizationOwner}
            />
          </TabsContent>

          <TabsContent value="members">
            {organizationQuery.error instanceof AxiosError &&
            organizationQuery.error.response?.status === 403 ? (
              <div className="py-8 text-center text-sm text-muted-foreground">
                You must be an organization owner to view members.
              </div>
            ) : (
              <div className="space-y-6">
                <div>
                  {isOrganizationOwner && (
                    <div className="mb-4 flex justify-end">
                      <Button
                        onClick={() =>
                          globalEmitter.emit('create-organization-invite', {
                            organizationId: orgId,
                            organizationName,
                          })
                        }
                      >
                        Invite Member
                      </Button>
                    </div>
                  )}
                  {organization?.members && organization.members.length > 0 ? (
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
                    <SimpleTable
                      data={pendingInvites}
                      columns={inviteColumns}
                    />
                  </div>
                )}
              </div>
            )}
          </TabsContent>

          <TabsContent value="tokens">
            {managementTokensQuery.error instanceof AxiosError &&
            managementTokensQuery.error.response?.status === 403 ? (
              <div className="py-8 text-center text-sm text-muted-foreground">
                You must be an organization owner to view management tokens.
              </div>
            ) : (
              <>
                {isOrganizationOwner && (
                  <div className="mb-4 flex justify-end">
                    <Button onClick={() => setShowCreateTokenModal(true)}>
                      Create Token
                    </Button>
                  </div>
                )}
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
              </>
            )}
          </TabsContent>
        </Tabs>
      </div>

      {isOrganizationOwner && memberToDelete && (
        <DeleteMemberModal
          open={!!memberToDelete}
          onOpenChange={(open) => !open && setMemberToDelete(null)}
          member={memberToDelete}
          organizationName={organizationName}
          onSuccess={() => organizationQuery.refetch()}
        />
      )}

      {isOrganizationOwner && (
        <CreateTokenModal
          open={showCreateTokenModal}
          onOpenChange={setShowCreateTokenModal}
          organizationId={orgId}
          organizationName={organizationName}
          onSuccess={() => managementTokensQuery.refetch()}
        />
      )}

      {isOrganizationOwner && tokenToDelete && (
        <DeleteTokenModal
          open={!!tokenToDelete}
          onOpenChange={(open) => !open && setTokenToDelete(null)}
          token={tokenToDelete}
          organizationName={organizationName}
          onSuccess={() => managementTokensQuery.refetch()}
        />
      )}

      {isOrganizationOwner && inviteToCancel && (
        <CancelInviteModal
          open={!!inviteToCancel}
          onOpenChange={(open) => !open && setInviteToCancel(null)}
          invite={inviteToCancel}
          organizationName={organizationName}
          onSuccess={() => organizationInvitesQuery.refetch()}
        />
      )}

      {tenantToArchive &&
        visibleTenants.find((t) => t?.id === tenantToArchive.id) && (
          <DeleteTenantModal
            open={!!tenantToArchive}
            onOpenChange={(open) => !open && setTenantToArchive(null)}
            tenant={tenantToArchive}
            tenantName={
              visibleTenants.find((t) => t?.id === tenantToArchive.id)?.name ||
              tenantToArchive.id
            }
            organizationName={organizationName}
            onSuccess={() => {
              queryClient.invalidateQueries({
                queryKey: ['organization:get', orgId],
              });
              queryClient.invalidateQueries({ queryKey: ['tenant:get'] });
            }}
          />
        )}
    </div>
  );
}

function OssOrganizationSettings() {
  const { tenantMemberships } = useUserUniverse();
  const queryClient = useQueryClient();

  const [tenantToArchive, setTenantToArchive] =
    useState<OrganizationTenant | null>(null);
  const [expandedTenantIds, setExpandedTenantIds] = useState<string[]>([]);

  const visibleTenants = useMemo(
    () =>
      tenantMemberships
        ?.map((m): OrganizationTenant | null => {
          if (!m.tenant) {
            return null;
          }
          return {
            id: m.tenant.metadata.id,
            name: m.tenant.name,
            status: TenantStatusType.ACTIVE,
            slug: m.tenant.slug,
          };
        })
        .filter((t): t is OrganizationTenant => t !== null) || [],
    [tenantMemberships],
  );

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Organization settings"
          description="Review the tenants associated with this workspace organization."
        />

        <TenantsSection
          tenants={visibleTenants}
          expandedTenantIds={expandedTenantIds}
          setExpandedTenantIds={setExpandedTenantIds}
          onArchive={setTenantToArchive}
          canManageOrganization={false}
        />
      </div>

      {tenantToArchive &&
        visibleTenants.find((t) => t?.id === tenantToArchive.id) && (
          <DeleteTenantModal
            open={!!tenantToArchive}
            onOpenChange={(open) => !open && setTenantToArchive(null)}
            tenant={tenantToArchive}
            tenantName={
              visibleTenants.find((t) => t?.id === tenantToArchive.id)?.name ||
              tenantToArchive.id
            }
            organizationName=""
            onSuccess={() => {
              queryClient.invalidateQueries({ queryKey: ['tenant:get'] });
            }}
          />
        )}
    </div>
  );
}

function formatExpiry(expiresAt?: string) {
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
}

function TenantsSection({
  tenants,
  expandedTenantIds,
  setExpandedTenantIds,
  onArchive,
  defaultOrganizationId,
  canManageOrganization,
}: {
  tenants: OrganizationTenant[];
  expandedTenantIds: string[];
  setExpandedTenantIds: (tenantIds: string[]) => void;
  onArchive: (tenant: OrganizationTenant) => void;
  defaultOrganizationId?: string;
  canManageOrganization: boolean;
}) {
  return (
    <>
      {canManageOrganization && (
        <div className="mb-4 flex justify-end">
          <Button
            onClick={() =>
              globalEmitter.emit('create-new-tenant', {
                defaultOrganizationId,
              })
            }
          >
            Add Tenant
          </Button>
        </div>
      )}
      {tenants.length ? (
        <Accordion
          type="multiple"
          value={expandedTenantIds}
          onValueChange={setExpandedTenantIds}
          className="space-y-3 rounded-md border bg-background p-3"
        >
          {tenants.map((tenant) => (
            <>
              <TenantAccordionItem
                key={tenant.id}
                tenant={tenant}
                isExpanded={expandedTenantIds.includes(tenant.id)}
                onArchive={onArchive}
                canManageOrganization={canManageOrganization}
              />
              <Separator className="my-3 last:hidden" />
            </>
          ))}
        </Accordion>
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No tenants found.
        </div>
      )}
    </>
  );
}

function TenantAccordionItem({
  tenant,
  isExpanded,
  onArchive,
  canManageOrganization,
}: {
  tenant: OrganizationTenant;
  isExpanded: boolean;
  onArchive: (tenant: OrganizationTenant) => void;
  canManageOrganization: boolean;
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
            canManageOrganization={canManageOrganization}
          />
        </div>
      </div>

      <AccordionContent className="border-border/70 px-4 pb-4 pt-4">
        <div className="space-y-5">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-medium">Members</h4>
            {canManageOrganization && (
              <Button
                onClick={() =>
                  globalEmitter.emit('create-tenant-invite', {
                    tenantId: tenant.id,
                  })
                }
              >
                Invite Member
              </Button>
            )}
          </div>

          {membersQuery.isLoading ? (
            <div className="py-4 text-sm text-muted-foreground">
              Loading members...
            </div>
          ) : membersQuery.isError &&
            membersQuery.error instanceof AxiosError &&
            [401, 403].includes(membersQuery.error.response?.status ?? 0) ? (
            <div className="py-4 text-sm text-muted-foreground">
              You must be a tenant admin or owner to view members.
            </div>
          ) : tenantMembers.length > 0 ? (
            <TenantMemberList
              tenantId={tenant.id}
              members={tenantMembers}
              canManage={canManageOrganization}
              onMembersChanged={() => membersQuery.refetch()}
            />
          ) : (
            <div className="py-4 text-sm text-muted-foreground">
              No members found.
            </div>
          )}

          {(invitesQuery.isLoading || tenantInvites.length > 0) && (
            <div className="space-y-2">
              <h4 className="text-sm font-medium">Pending Invites</h4>
              <TenantInviteList invites={tenantInvites} />
            </div>
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  );
}

function TenantMemberList({
  tenantId,
  members,
  canManage,
  onMembersChanged,
}: {
  tenantId: string;
  members: TenantMember[];
  canManage: boolean;
  onMembersChanged: () => void;
}) {
  const [memberToEdit, setMemberToEdit] = useState<TenantMember | null>(null);

  const gridCols = canManage
    ? 'grid-cols-[minmax(0,1.2fr)_minmax(0,1.6fr)_140px_140px_72px]'
    : 'grid-cols-[minmax(0,1.2fr)_minmax(0,1.6fr)_140px_140px]';

  return (
    <>
      <div className="rounded-md border border-border/70">
        <div
          className={`hidden ${gridCols} gap-3 border-b border-border/70 bg-muted/20 px-4 py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground md:grid`}
        >
          <span>Name</span>
          <span>Email</span>
          <span>Role</span>
          <span>Joined</span>
          {canManage && <span className="text-right">Actions</span>}
        </div>
        <div>
          {members.map((member) => (
            <div
              key={member.metadata.id}
              className={`grid gap-3 border-b border-border/50 px-4 py-3 last:border-b-0 md:${gridCols} md:items-center`}
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
                <p className="text-xs text-muted-foreground md:hidden">
                  Joined
                </p>
                <RelativeDate date={member.metadata.createdAt} />
              </div>
              {canManage && (
                <div className="flex justify-end">
                  <TenantMemberActions
                    member={member}
                    tenantId={tenantId}
                    onEditRoleClick={setMemberToEdit}
                    onChangePasswordClick={() => {}}
                    onDeleteSuccess={onMembersChanged}
                  />
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {memberToEdit && (
        <TenantMemberUpdateDialog
          tenantId={tenantId}
          member={memberToEdit}
          onClose={() => setMemberToEdit(null)}
          onSuccess={() => {
            setMemberToEdit(null);
            onMembersChanged();
          }}
        />
      )}
    </>
  );
}

function TenantMemberUpdateDialog({
  tenantId,
  member,
  onClose,
  onSuccess,
}: {
  tenantId: string;
  member: TenantMember;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors,
  });
  const { tenantMemberUpdateMutation } = useTenantApi();
  const memberUpdate = tenantMemberUpdateMutation(tenantId, member.metadata.id);
  const updateMutation = useMutation({
    ...memberUpdate,
    mutationFn: async (data: { role: TenantMemberRole }) => {
      if (data.role === TenantMemberRole.OWNER) {
        throw new Error(
          'OWNER role management must be done through organization membership',
        );
      }

      await memberUpdate.mutationFn(data);
    },
    onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <UpdateMemberForm
        isLoading={updateMutation.isPending}
        onSubmit={(data) => updateMutation.mutate(data)}
        fieldErrors={fieldErrors}
        member={member}
        isCloudEnabled={true}
      />
    </Dialog>
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
  canManageOrganization,
}: {
  row: OrganizationTenant & { metadata: { id: string } };
  onArchive: (tenant: OrganizationTenant) => void;
  canManageOrganization: boolean;
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
        {canManageOrganization && (
          <DropdownMenuItem
            onClick={() => onArchive({ id: row.id, status: row.status })}
          >
            <TrashIcon className="mr-2 size-4" />
            Archive Tenant
          </DropdownMenuItem>
        )}
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
