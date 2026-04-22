import { CancelInviteModal } from './components/cancel-invite-modal';
import { CreateTokenModal } from './components/create-token-modal';
import { DeleteMemberModal } from './components/delete-member-modal';
import { DeleteTenantModal } from './components/delete-tenant-modal';
import { DeleteTokenModal } from './components/delete-token-modal';
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
import { Loading } from '@/components/v1/ui/loading';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useOrganizations } from '@/hooks/use-organizations';
import {
  OrganizationMember,
  ManagementToken,
  OrganizationInvite,
  OrganizationInviteStatus,
  TenantStatusType,
  OrganizationTenant,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { lastTenantAtom } from '@/lib/atoms';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta.ts';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import SSOPage from '@/pages/organizations/$organization/components/sso-setup.tsx';
import { useUserUniverse } from '@/providers/user-universe.tsx';
import { appRoutes } from '@/router';
import {
  PlusIcon,
  BuildingOffice2Icon,
  UserIcon,
  KeyIcon,
  PencilIcon,
  CheckIcon,
  XMarkIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
} from '@heroicons/react/24/outline';
import { EllipsisVerticalIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { formatDistanceToNow } from 'date-fns';
import { useAtomValue } from 'jotai';
import { useState } from 'react';

type Section = 'tenants' | 'members' | 'tokens' | 'sso';

const NAV_ITEMS: Record<
  string,
  { key: Section; label: string; icon: typeof KeyIcon }
> = {
  tenants: { key: 'tenants', label: 'Tenants', icon: BuildingOffice2Icon },
  members: { key: 'members', label: 'Members', icon: UserIcon },
  tokens: { key: 'tokens', label: 'Management Tokens', icon: KeyIcon },
};

export default function OrganizationPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const {
    handleUpdateOrganization,
    updateOrganizationLoading,
    handleCreateOrganizationSsoDomain,
    handleDeleteOrganizationSsoDomain,
  } = useOrganizations();
  const orgApi = useOrganizationApi();
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
  const lastTenant = useAtomValue(lastTenantAtom);
  const [activeSection, setActiveSection] = useState<Section>('tenants');
  const [isEditingName, setIsEditingName] = useState(false);
  const [editedName, setEditedName] = useState('');
  const { meta } = useApiMeta();
  const schemes = meta?.auth?.schemes || [];
  // conditionally enable the SSO button
  if (schemes.includes('sso')) {
    NAV_ITEMS.sso = { key: 'sso', label: 'SSO', icon: KeyIcon };
  }
  const [newSsoDomain, setNewSsoDomain] = useState('');
  const [isAddingSsoDomain, setIsAddingSsoDomain] = useState(false);

  const { isCloudEnabled } = useUserUniverse();
  const handleStartEdit = () => {
    if (organizationQuery.data?.name) {
      setEditedName(organizationQuery.data.name);
      setIsEditingName(true);
    }
  };

  const handleCancelEdit = () => {
    setIsEditingName(false);
    setEditedName('');
  };

  const handleSaveEdit = () => {
    if (!orgId || !editedName.trim()) {
      return;
    }

    handleUpdateOrganization(orgId, editedName.trim(), () => {
      setIsEditingName(false);
      setEditedName('');
      queryClient.invalidateQueries({ queryKey: ['organization:get', orgId] });
    });
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSaveEdit();
    } else if (e.key === 'Escape') {
      handleCancelEdit();
    }
  };

  const formatExpirationDate = (expiresDate?: string) => {
    if (!expiresDate) {
      return 'never';
    }

    try {
      const expires = new Date(expiresDate);
      const now = new Date();

      if (expires < now) {
        return 'expired';
      }

      return `in ${formatDistanceToNow(expires)}`;
    } catch {
      return new Date(expiresDate).toLocaleDateString();
    }
  };

  const organizationQuery = useQuery({
    ...orgApi.organizationGetQuery(orgId),
    enabled: !!orgId,
  });

  const managementTokensQuery = useQuery({
    ...orgApi.managementTokenListQuery(orgId),
    enabled: !!orgId,
  });

  const organizationInvitesQuery = useQuery({
    ...orgApi.organizationInviteListQuery(orgId),
    enabled: !!orgId,
  });

  const organizationSsoDomainGetQuery = useQuery({
    ...orgApi.organizationSsoDomainGetQuery(orgId),
    enabled: !!orgId,
  });

  const handleAddSsoDomain = async () => {
    if (!orgId || !newSsoDomain.trim()) {
      return;
    }
    setIsAddingSsoDomain(true);
    handleCreateOrganizationSsoDomain(
      orgId,
      newSsoDomain.trim(),
      () => {
        setIsAddingSsoDomain(false);
        setNewSsoDomain('');
        queryClient.invalidateQueries({
          queryKey: ['organization:sso_domain:get', orgId],
        });
      },
      () => {
        setIsAddingSsoDomain(false);
      },
    );
  };

  const handleDeleteSsoDomain = async (domain: string) => {
    if (!orgId) {
      return;
    }
    handleDeleteOrganizationSsoDomain(orgId, domain, () => {
      queryClient.invalidateQueries({
        queryKey: ['organization:sso_domain:get', orgId],
      });
    });
  };

  const { currentUser } = useCurrentUser();

  if (organizationQuery.isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-background">
        <Loading />
      </div>
    );
  }

  if (organizationQuery.isError) {
    const status = isAxiosError(organizationQuery.error)
      ? organizationQuery.error.response?.status
      : undefined;

    if (status === 404 || status === 403) {
      return (
        <ResourceNotFound
          resource="Organization"
          description="The organization you're looking for doesn't exist or you don't have access to it."
          primaryAction={{
            label: 'Dashboard',
            navigate: { to: appRoutes.authenticatedRoute.to },
          }}
        />
      );
    }

    throw organizationQuery.error;
  }

  if (!organizationQuery.data) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-background">
        <Loading />
      </div>
    );
  }

  const organization = organizationQuery.data;

  const handleGoBack = () => {
    const tenantId = lastTenant?.metadata.id ?? organization.tenants?.[0]?.id;
    if (tenantId) {
      navigate({
        to: appRoutes.tenantRunsRoute.to,
        params: { tenant: tenantId },
      });
    } else {
      navigate({ to: '/' });
    }
  };

  const tenantColumns = [
    {
      columnLabel: 'Name',
      cellRenderer: (
        row: OrganizationTenant & { metadata: { id: string } },
      ) => {
        return <span className="font-medium">{row.name || row.id}</span>;
      },
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
      ) => {
        return <span className="text-muted-foreground">{row.slug || '-'}</span>;
      },
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (
        row: OrganizationTenant & { metadata: { id: string } },
      ) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
              <EllipsisVerticalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() => {
                navigate({
                  to: appRoutes.tenantRoute.to,
                  params: { tenant: row.id },
                });
              }}
            >
              <ArrowRightIcon className="mr-2 size-4" />
              View Tenant
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() =>
                setTenantToArchive({ id: row.id, status: row.status })
              }
            >
              <TrashIcon className="mr-2 size-4" />
              Archive Tenant
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
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
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
              <EllipsisVerticalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {currentUser?.email === row.email ? (
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
              <DropdownMenuItem onClick={() => setMemberToDelete(row)}>
                <TrashIcon className="mr-2 size-4" />
                Remove Member
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
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
        <span>{formatExpirationDate(row.expires)}</span>
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
        <span>{formatExpirationDate(row.expiresAt)}</span>
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

  // SSO domains table columns
  const ssoDomainColumns = [
    {
      columnLabel: 'Domain',
      cellRenderer: (row: {
        metadata: { id: string };
        domain: string;
        verified?: boolean;
        verification_token?: string;
      }) => <span className="font-mono text-sm">{row.domain}</span>,
    },
    {
      columnLabel: 'Verified',
      cellRenderer: (row: {
        metadata: { id: string };
        domain: string;
        verified?: boolean;
        verification_token?: string;
      }) => (
        <Badge variant={row.verified ? 'successful' : 'secondary'}>
          {row.verified ? 'Verified' : 'Unverified'}
        </Badge>
      ),
    },
    {
      columnLabel: 'Verification Token',
      cellRenderer: (row: {
        metadata: { id: string };
        domain: string;
        verified?: boolean;
        verification_token?: string;
      }) =>
        row.verification_token ? (
          <div className="flex items-center gap-2">
            <span className="font-mono text-xs text-muted-foreground">
              {row.verification_token}
            </span>
            <CopyToClipboard text={row.verification_token} />
          </div>
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (row: {
        metadata: { id: string };
        domain: string;
        verified?: boolean;
        verification_token?: string;
      }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
              <EllipsisVerticalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => handleDeleteSsoDomain(row.domain)}>
              <TrashIcon className="mr-2 size-4" />
              Remove Domain
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  const pendingInvites = organizationInvitesQuery.data?.rows?.filter(
    (invite) =>
      invite.status === OrganizationInviteStatus.PENDING ||
      invite.status === OrganizationInviteStatus.EXPIRED,
  );
  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      <div className="flex h-14 shrink-0 items-center justify-between border-b px-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleGoBack}
          className="gap-2 text-muted-foreground"
        >
          <ArrowLeftIcon className="size-4" />
          Back to Dashboard
        </Button>
        <div className="flex items-center">
          {isEditingName ? (
            <div className="flex items-center gap-1.5">
              <Input
                value={editedName}
                onChange={(e) => setEditedName(e.target.value)}
                onKeyDown={handleKeyPress}
                className="h-8 w-48 px-2 text-sm font-semibold"
                autoFocus
                disabled={updateOrganizationLoading}
              />
              <Button
                size="sm"
                variant="ghost"
                className="h-7 w-7 shrink-0 p-0"
                onClick={handleSaveEdit}
                disabled={updateOrganizationLoading || !editedName.trim()}
              >
                <CheckIcon className="size-3.5" />
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="h-7 w-7 shrink-0 p-0"
                onClick={handleCancelEdit}
                disabled={updateOrganizationLoading}
              >
                <XMarkIcon className="size-3.5" />
              </Button>
            </div>
          ) : (
            <Button
              variant="ghost"
              size="lg"
              onClick={handleStartEdit}
              disabled={updateOrganizationLoading}
              className="gap-x-3 text-lg font-semibold px-4"
            >
              {organization.name}
              <PencilIcon className="size-4 text-muted-foreground" />
            </Button>
          )}
        </div>
      </div>

      <div className="grid flex-1 grid-cols-[240px_1fr] overflow-hidden">
        <div className="flex flex-col border-r">
          <nav className="flex-1 space-y-1 px-3 py-3">
            {Object.values(NAV_ITEMS).map((item) => (
              <button
                key={item.key}
                onClick={() => setActiveSection(item.key)}
                className={cn(
                  'flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  activeSection === item.key
                    ? 'bg-muted text-foreground'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                )}
              >
                <item.icon className="size-4 shrink-0" />
                {item.label}
              </button>
            ))}
          </nav>
        </div>

        <div className="flex flex-col overflow-hidden">
          <div className="flex h-12 shrink-0 items-center justify-between p-8">
            <h2 className="text-lg font-semibold">
              {NAV_ITEMS[activeSection]?.label}
            </h2>
            {activeSection === 'tenants' && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  globalEmitter.emit('create-new-tenant', {
                    defaultOrganizationId: organization.metadata.id,
                  });
                }}
                leftIcon={<PlusIcon className="size-4" />}
              >
                Add Tenant
              </Button>
            )}
            {activeSection === 'members' && (
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  globalEmitter.emit('create-organization-invite', {
                    organizationId: orgId,
                    organizationName: organization.name,
                  })
                }
                leftIcon={<PlusIcon className="size-4" />}
              >
                Invite Member
              </Button>
            )}
            {activeSection === 'tokens' && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowCreateTokenModal(true)}
                leftIcon={<PlusIcon className="size-4" />}
              >
                Create Token
              </Button>
            )}
            {activeSection === 'sso' && isCloudEnabled && <SSOPage />}
          </div>

          <div className="flex-1 overflow-y-auto px-8">
            {activeSection === 'tenants' && (
              <>
                {organization.tenants && organization.tenants.length > 0 ? (
                  <SimpleTable
                    data={organization.tenants
                      .filter(
                        (tenant) => tenant.status !== TenantStatusType.ARCHIVED,
                      )
                      .map((t) => ({ ...t, metadata: { id: t.id } }))}
                    columns={tenantColumns}
                  />
                ) : (
                  <div className="py-16 text-center">
                    <BuildingOffice2Icon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                    <h3 className="mb-2 text-lg font-medium">No Tenants Yet</h3>
                    <p className="mb-4 text-muted-foreground">
                      Add your first tenant to get started.
                    </p>
                    <Button
                      onClick={() => {
                        globalEmitter.emit('create-new-tenant', {
                          defaultOrganizationId: organization.metadata.id,
                        });
                      }}
                      leftIcon={<PlusIcon className="size-4" />}
                    >
                      Add Tenant
                    </Button>
                  </div>
                )}
              </>
            )}

            {activeSection === 'members' && (
              <div className="space-y-8">
                {organization.members && organization.members.length > 0 ? (
                  <SimpleTable
                    data={organization.members}
                    columns={memberColumns}
                  />
                ) : (
                  <div className="py-16 text-center">
                    <UserIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                    <h3 className="mb-2 text-lg font-medium">No Members Yet</h3>
                    <p className="mb-4 text-muted-foreground">
                      Members will appear here when they join this organization.
                    </p>
                  </div>
                )}

                {organizationInvitesQuery.isLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <Loading />
                  </div>
                ) : pendingInvites && pendingInvites.length > 0 ? (
                  <div className="space-y-3">
                    <h3 className="text-sm font-medium text-muted-foreground">
                      Pending Invites
                    </h3>
                    <SimpleTable
                      data={pendingInvites}
                      columns={inviteColumns}
                    />
                  </div>
                ) : null}
              </div>
            )}

            {activeSection === 'tokens' && (
              <>
                {managementTokensQuery.isLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <Loading />
                  </div>
                ) : managementTokensQuery.data &&
                  managementTokensQuery.data.rows &&
                  managementTokensQuery.data.rows.length > 0 ? (
                  <SimpleTable
                    data={managementTokensQuery.data.rows.map((t) => ({
                      ...t,
                      metadata: { id: t.id },
                    }))}
                    columns={tokenColumns}
                  />
                ) : (
                  <div className="py-16 text-center">
                    <KeyIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                    <h3 className="mb-2 text-lg font-medium">
                      No Management Tokens
                    </h3>
                    <p className="mb-4 text-muted-foreground">
                      Create API tokens to manage this organization
                      programmatically.
                    </p>
                    <Button
                      onClick={() => setShowCreateTokenModal(true)}
                      leftIcon={<PlusIcon className="size-4" />}
                    >
                      Create Token
                    </Button>
                  </div>
                )}
              </>
            )}

            {activeSection === 'sso' && schemes.includes('sso') && (
              <div className="space-y-8">
                {/* SSO Domains Table */}
                {organizationSsoDomainGetQuery.isLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <Loading />
                  </div>
                ) : organizationSsoDomainGetQuery.data &&
                  organizationSsoDomainGetQuery.data.length > 0 ? (
                  <SimpleTable
                    data={organizationSsoDomainGetQuery.data.map((v) => {
                      return {
                        metadata: { id: v.ssoDomain },
                        domain: v.ssoDomain,
                        verified: v.verified,
                        verification_token: v.verificationToken,
                      };
                    })}
                    columns={ssoDomainColumns}
                  />
                ) : (
                  <div className="py-16 text-center">
                    <KeyIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                    <h3 className="mb-2 text-lg font-medium">No SSO Domains</h3>
                    <p className="mb-4 text-muted-foreground">
                      Add a domain below to enable SSO for your organization.
                    </p>
                  </div>
                )}

                {organizationSsoDomainGetQuery.data &&
                  organizationSsoDomainGetQuery.data.length > 0 && (
                    <div className="rounded-md border border-muted bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
                      <p>
                        To verify your domain, add a DNS TXT record with the
                        value:
                      </p>
                      <p className="mt-1 font-mono">
                        hatchet-sso-verify=&#123;verification_token&#125;
                      </p>
                      <p className="mt-2">
                        It may take a few minutes for DNS changes to propagate
                        and for the verified status to update.
                      </p>
                    </div>
                  )}

                {/* Add New SSO Domain */}
                <div className="space-y-2">
                  <h3 className="text-sm font-medium text-muted-foreground">
                    Add Domain
                  </h3>
                  <div className="flex gap-2">
                    <Input
                      placeholder="example.com"
                      value={newSsoDomain}
                      onChange={(e) => setNewSsoDomain(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          handleAddSsoDomain();
                        }
                      }}
                      className="max-w-sm"
                      disabled={isAddingSsoDomain}
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleAddSsoDomain}
                      disabled={isAddingSsoDomain || !newSsoDomain.trim()}
                      leftIcon={<PlusIcon className="size-4" />}
                    >
                      Add Domain
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {memberToDelete && organization && (
        <DeleteMemberModal
          open={!!memberToDelete}
          onOpenChange={(open) => !open && setMemberToDelete(null)}
          member={memberToDelete}
          organizationName={organization.name}
          onSuccess={() => {
            organizationQuery.refetch();
          }}
        />
      )}

      {orgId && organization && (
        <CreateTokenModal
          open={showCreateTokenModal}
          onOpenChange={setShowCreateTokenModal}
          organizationId={orgId}
          organizationName={organization.name}
          onSuccess={() => {
            managementTokensQuery.refetch();
          }}
        />
      )}

      {tokenToDelete && organization && (
        <DeleteTokenModal
          open={!!tokenToDelete}
          onOpenChange={(open) => !open && setTokenToDelete(null)}
          token={tokenToDelete}
          organizationName={organization.name}
          onSuccess={() => {
            managementTokensQuery.refetch();
          }}
        />
      )}

      {inviteToCancel && organization && (
        <CancelInviteModal
          open={!!inviteToCancel}
          onOpenChange={(open) => !open && setInviteToCancel(null)}
          invite={inviteToCancel}
          organizationName={organization.name}
          onSuccess={() => {
            organizationInvitesQuery.refetch();
          }}
        />
      )}

      {(() => {
        const foundTenant = tenantToArchive
          ? organization.tenants?.find((t) => t?.id === tenantToArchive.id)
          : undefined;

        if (!foundTenant) {
          return null;
        }

        return (
          tenantToArchive &&
          organization &&
          foundTenant && (
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
          )
        );
      })()}
    </div>
  );
}
