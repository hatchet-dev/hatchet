import { SettingsPageHeader } from '../components/settings-page-header';
import { usePylon } from '@/components/support-chat';
import { TenantRegionBadge } from '@/components/v1/molecules/nav-bar/tenant-region-badge';
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
import { Loading } from '@/components/v1/ui/loading';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { Switch } from '@/components/v1/ui/switch';
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
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentUser } from '@/hooks/use-current-user';
import { FeatureFlagId, useIsFeatureEnabled } from '@/hooks/use-feature-flags';
import { useOrganizations } from '@/hooks/use-organizations';
import { TenantInvite, TenantMember, TenantMemberRole } from '@/lib/api';
import {
  ManagementToken,
  OrganizationInvite,
  OrganizationInviteStatus,
  OrganizationMember,
  OrganizationTenant,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  OrganizationAvailableShard,
  OrganizationAvailableShardClass,
  OrganizationTenant as ControlPlaneOrganizationTenant,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { globalEmitter } from '@/lib/global-emitter';
import { useApiError } from '@/lib/hooks';
import { parseDuration, msToDurationString } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta.ts';
import { MemberActions as TenantMemberActions } from '@/pages/main/v1/tenant-settings/members/components/members-columns';
import { UpdateMemberForm } from '@/pages/main/v1/tenant-settings/members/components/update-member-form';
import CreateSSOPage from '@/pages/main/v1/tenant-settings/organization/components/sso-setup.tsx';
import { CancelInviteModal } from '@/pages/organizations/$organization/components/cancel-invite-modal';
import { CreateTokenModal } from '@/pages/organizations/$organization/components/create-token-modal';
import { DeleteMemberModal } from '@/pages/organizations/$organization/components/delete-member-modal';
import { DeleteTenantModal } from '@/pages/organizations/$organization/components/delete-tenant-modal';
import { DeleteTokenModal } from '@/pages/organizations/$organization/components/delete-token-modal';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import {
  PlusIcon,
  KeyIcon,
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
import { useEffect, useMemo, useRef, useState } from 'react';

const OFFICE_HOURS_URL = 'https://hatchet.run/office-hours';

function formatTimeoutMs(ms: number): string {
  if (ms <= 0) {
    return 'Disabled';
  }
  const minutes = Math.floor(ms / 60000);
  if (minutes < 60) {
    return `${minutes} minute${minutes !== 1 ? 's' : ''}`;
  }
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  if (hours < 24) {
    if (remMinutes === 0) {
      return `${hours} hour${hours !== 1 ? 's' : ''}`;
    }
    return `${hours} hour${hours !== 1 ? 's' : ''} ${remMinutes} minute${remMinutes !== 1 ? 's' : ''}`;
  }
  const days = Math.floor(hours / 24);
  const remHours = hours % 24;
  if (remHours === 0) {
    return `${days} day${days !== 1 ? 's' : ''}`;
  }
  return `${days} day${days !== 1 ? 's' : ''} ${remHours} hour${remHours !== 1 ? 's' : ''}`;
}

// FIXME: remove this once we migrate everyone to the control plane
type OrganizationTenantWithRegion = OrganizationTenant & {
  region?: ControlPlaneOrganizationTenant['region'];
  canManage?: boolean;
};

export function CloudOrganizationSettings({ orgId }: { orgId: string }) {
  const { isControlPlaneEnabled } = useControlPlane();
  const pylon = usePylon();
  const {
    organizations,
    handleUpdateOrganization,
    handleUpdateOrganizationTimeout,
    updateOrganizationLoading,
    handleCreateOrganizationSsoDomain,
    handleDeleteOrganizationSsoDomain,
  } = useOrganizations();

  const org = organizations.find((o) => o.metadata.id === orgId);
  const isOrganizationOwner = org?.isOwner ?? false;

  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { currentUser } = useCurrentUser();
  const { meta } = useApiMeta();
  const schemes = meta?.auth?.schemes || [];
  const { isEnabled: organizationSsoEnabled } = useIsFeatureEnabled(
    FeatureFlagId.OrganizationSsoEnabled,
    false,
  );
  const canManageSso =
    isOrganizationOwner && schemes.includes('sso') && organizationSsoEnabled;
  const [memberToDelete, setMemberToDelete] =
    useState<OrganizationMember | null>(null);
  const [showCreateTokenModal, setShowCreateTokenModal] = useState(false);
  const [tokenToDelete, setTokenToDelete] = useState<ManagementToken | null>(
    null,
  );
  const [inviteToCancel, setInviteToCancel] =
    useState<OrganizationInvite | null>(null);
  const [tenantToArchive, setTenantToArchive] =
    useState<OrganizationTenantWithRegion | null>(null);
  const [expandedTenantIds, setExpandedTenantIds] = useState<string[]>([]);
  const autoExpandedTenantId = useRef<string | null>(null);
  const [editedName, setEditedName] = useState('');
  const [isEditingName, setIsEditingName] = useState(false);
  const [editedTimeout, setEditedTimeout] = useState('');
  const [isEditingTimeout, setIsEditingTimeout] = useState(false);
  const [newSsoDomain, setNewSsoDomain] = useState('');
  const [isAddingSsoDomain, setIsAddingSsoDomain] = useState(false);

  const organizationSsoDomainGetQuery = useQuery({
    ...orgApi.organizationSsoDomainGetQuery(orgId),
    enabled: !!orgId && canManageSso,
  });

  const organizationSsoConfigGetQuery = useQuery({
    ...orgApi.organizationSsoConfigGetQuery(orgId),
    enabled: !!orgId && canManageSso,
  });

  const ssoConfigUpdateMutation = useMutation({
    ...orgApi.organizationSsoConfigUpdateMutation(orgId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization:sso_config:get', orgId],
      });
    },
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

  const organizationAvailableShardsQuery = useQuery({
    ...orgApi.organizationAvailableShardsQuery(orgId!),
    enabled: !!orgId && isControlPlaneEnabled && isOrganizationOwner,
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

  // showing the first tenant as open, to make clearer that:
  // 1. tenants can expand
  // 2. you can add members to tenants from here
  useEffect(() => {
    const firstVisibleTenantId = visibleTenants[0]?.id;

    if (!firstVisibleTenantId) {
      autoExpandedTenantId.current = null;
      return;
    }

    if (autoExpandedTenantId.current === firstVisibleTenantId) {
      return;
    }

    autoExpandedTenantId.current = firstVisibleTenantId;

    if (!expandedTenantIds.length) {
      setExpandedTenantIds([firstVisibleTenantId]);
    }
  }, [visibleTenants, expandedTenantIds.length]);

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

  const cpOrganization = isControlPlaneEnabled
    ? (organization as { inactivity_timeout?: number } | undefined)
    : undefined;
  const currentInactivityTimeoutMs = cpOrganization?.inactivity_timeout ?? -1;

  const parsedEditedTimeout = useMemo(
    () => parseDuration(editedTimeout),
    [editedTimeout],
  );

  const handleSaveTimeout = () => {
    if (!orgId || parsedEditedTimeout === null) {
      return;
    }
    if (parsedEditedTimeout === currentInactivityTimeoutMs) {
      setIsEditingTimeout(false);
      return;
    }
    handleUpdateOrganizationTimeout(orgId, parsedEditedTimeout, () => {
      setIsEditingTimeout(false);
      queryClient.invalidateQueries({ queryKey: ['organization:get', orgId] });
    });
  };

  const handleStartEditingTimeout = () => {
    setEditedTimeout(msToDurationString(currentInactivityTimeoutMs));
    setIsEditingTimeout(true);
  };

  const handleCancelEditingTimeout = () => {
    setIsEditingTimeout(false);
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

  // SSO domains table columns
  const ssoDomainColumns = [
    {
      columnLabel: 'Domain',
      cellRenderer: (row: {
        domain: string;
        verified?: boolean;
        verification_token?: string;
      }) => <span className="font-mono text-sm">{row.domain}</span>,
    },
    {
      columnLabel: 'Verified',
      cellRenderer: (row: {
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

  const availableShardColumns = [
    {
      columnLabel: 'Cloud Provider',
      cellRenderer: (row: OrganizationAvailableShard) =>
        row.provider ? (
          <Badge variant="outline">{row.provider}</Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      columnLabel: 'Region',
      cellRenderer: (row: OrganizationAvailableShard) => (
        <span className="font-mono text-sm">{row.region}</span>
      ),
    },
    {
      columnLabel: 'Class',
      cellRenderer: (row: OrganizationAvailableShard) =>
        row.shardClass === OrganizationAvailableShardClass.DEDICATED ? (
          <Badge variant="secondary">Dedicated</Badge>
        ) : (
          <Badge variant="outline">Shared</Badge>
        ),
    },
  ];

  const tokenColumns = [
    {
      columnLabel: 'Name',
      cellRenderer: (row: ManagementToken) => (
        <span className="font-medium">{row.name}</span>
      ),
    },
    {
      columnLabel: 'Expiry',
      cellRenderer: (row: ManagementToken) => (
        <span>{formatExpiry(row.expiresAt)}</span>
      ),
    },
    ...(isOrganizationOwner
      ? [
          {
            columnLabel: 'Actions',
            cellRenderer: (row: ManagementToken) => (
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
            <div className="flex flex-col gap-3">
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

              {isControlPlaneEnabled && (
                <div className="w-full rounded-lg border border-border/50 bg-muted/10 p-4 md:max-w-sm">
                  <div className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Inactivity timeout
                  </div>

                  {isEditingTimeout ? (
                    <div className="flex flex-col gap-1.5">
                      <div className="flex items-center gap-2">
                        <Input
                          type="text"
                          value={editedTimeout}
                          onChange={(e) => setEditedTimeout(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                              handleSaveTimeout();
                            }
                            if (e.key === 'Escape') {
                              handleCancelEditingTimeout();
                            }
                          }}
                          className="h-10 flex-1 bg-background/60"
                          placeholder="e.g. 30m, 1h, 1h30m, -1 to disable"
                          disabled={updateOrganizationLoading}
                          autoFocus
                        />
                        <div className="flex items-center gap-2">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={handleCancelEditingTimeout}
                            disabled={updateOrganizationLoading}
                            hoverText="Cancel editing"
                            className="shrink-0 hover:bg-muted/50"
                          >
                            <XMarkIcon className="size-4" />
                          </Button>
                          <Button
                            variant="outline"
                            size="icon"
                            onClick={handleSaveTimeout}
                            disabled={
                              updateOrganizationLoading ||
                              parsedEditedTimeout === null
                            }
                            hoverText="Save inactivity timeout"
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
                      {editedTimeout.trim() !== '' && (
                        <p
                          className={`text-xs ${parsedEditedTimeout === null ? 'text-destructive' : 'text-muted-foreground'}`}
                        >
                          {parsedEditedTimeout === null
                            ? 'Invalid format — try 30m, 1h, 1h30m, 100ms'
                            : `→ ${formatTimeoutMs(parsedEditedTimeout)}`}
                        </p>
                      )}
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <div className="flex h-10 min-w-0 flex-1 items-center rounded-md border border-input bg-background/60 px-3">
                        <p className="truncate text-sm font-medium text-foreground">
                          {formatTimeoutMs(currentInactivityTimeoutMs)}
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={handleStartEditingTimeout}
                        hoverText="Edit inactivity timeout"
                        className="shrink-0 bg-background/60 hover:bg-muted/50"
                      >
                        <PencilSquareIcon className="size-4" />
                      </Button>
                    </div>
                  )}
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
            {isOrganizationOwner && isControlPlaneEnabled && (
              <TabsTrigger value="regions" variant="underlined">
                Available Regions
              </TabsTrigger>
            )}
            {canManageSso && (
              <TabsTrigger value="sso" variant="underlined">
                SSO
              </TabsTrigger>
            )}
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
                      rowKey={(row) => row.metadata.id}
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
                      rowKey={(row) => row.metadata.id}
                    />
                  </div>
                )}
              </div>
            )}
          </TabsContent>

          {isOrganizationOwner && isControlPlaneEnabled && (
            <TabsContent value="regions">
              {organizationAvailableShardsQuery.isLoading ? (
                <div className="flex justify-center py-12">
                  <Loading />
                </div>
              ) : organizationAvailableShardsQuery.error instanceof
                  AxiosError &&
                organizationAvailableShardsQuery.error.response?.status ===
                  403 ? (
                <div className="py-8 text-center text-sm text-muted-foreground">
                  You must be an organization owner to view available regions.
                </div>
              ) : organizationAvailableShardsQuery.error ? (
                <div className="py-8 text-center text-sm text-destructive">
                  Failed to load available regions.
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="space-y-2 text-sm text-muted-foreground">
                    <p>
                      Regions where new tenants can be deployed for this
                      organization.
                    </p>
                    <p>
                      Need to configure which regions are available for a
                      tenant, or looking for a new region?{' '}
                      {pylon.enabled ? (
                        <>
                          <Button
                            type="button"
                            variant="link"
                            className="h-auto p-0 text-sm font-normal"
                            onClick={() => pylon.show()}
                          >
                            Open support chat
                          </Button>
                          , or{' '}
                        </>
                      ) : null}
                      <a
                        href={OFFICE_HOURS_URL}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary underline-offset-4 hover:underline"
                      >
                        Schedule office hours
                      </a>
                      .
                    </p>
                  </div>
                  {organizationAvailableShardsQuery.data?.rows &&
                  organizationAvailableShardsQuery.data.rows.length > 0 ? (
                    <SimpleTable
                      data={organizationAvailableShardsQuery.data.rows}
                      columns={availableShardColumns}
                      rowKey={(row) =>
                        `${row.shardClass}:${row.provider}:${row.region}`
                      }
                    />
                  ) : (
                    <div className="py-8 text-center text-sm text-muted-foreground">
                      No deployment regions are configured.
                    </div>
                  )}
                </div>
              )}
            </TabsContent>
          )}

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
                    data={managementTokensQuery.data.rows}
                    columns={tokenColumns}
                    rowKey={(row) => row.id}
                  />
                ) : (
                  <div className="py-8 text-center text-sm text-muted-foreground">
                    No management tokens found.
                  </div>
                )}
              </>
            )}
          </TabsContent>
          {canManageSso && (
            <TabsContent value="sso">
              <div className="space-y-8">
                <CreateSSOPage orgId={orgId} />
                {/* Force SSO toggle */}
                {isOrganizationOwner && (
                  <div className="flex items-center justify-between rounded-lg border border-border/50 bg-muted/10 p-4">
                    <div className="space-y-0.5">
                      <p className="text-sm font-medium">Force SSO</p>
                      <p className="text-sm text-muted-foreground">
                        Require all organization members to sign in with SSO.
                        All other login methods will be disabled.
                      </p>
                    </div>
                    <Switch
                      checked={
                        organizationSsoConfigGetQuery.data?.forceSSO ?? false
                      }
                      onCheckedChange={(checked) =>
                        ssoConfigUpdateMutation.mutate(checked)
                      }
                      disabled={
                        organizationSsoConfigGetQuery.isLoading ||
                        ssoConfigUpdateMutation.isPending
                      }
                    />
                  </div>
                )}
                {/* SSO Domains Table */}
                {organizationSsoDomainGetQuery.isLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <Loading />
                  </div>
                ) : organizationSsoDomainGetQuery.data &&
                  organizationSsoDomainGetQuery.data.length > 0 ? (
                  <SimpleTable
                    data={organizationSsoDomainGetQuery.data.map((v) => ({
                      domain: v.ssoDomain,
                      verified: v.verified,
                      verification_token: v.verificationToken,
                    }))}
                    columns={ssoDomainColumns}
                    rowKey={(row) => row.domain}
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

                {/* Add New SSO Domain */}
                <div className="space-y-2">
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
              </div>
            </TabsContent>
          )}
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

export function OssOrganizationSettings() {
  const { tenantMemberships } = useUserUniverse();
  const queryClient = useQueryClient();

  const [tenantToArchive, setTenantToArchive] =
    useState<OrganizationTenantWithRegion | null>(null);
  const [expandedTenantIds, setExpandedTenantIds] = useState<string[]>([]);
  const autoExpandedTenantId = useRef<string | null>(null);

  const visibleTenants = useMemo(
    () =>
      tenantMemberships
        ?.map((m): OrganizationTenantWithRegion | null => {
          if (!m.tenant) {
            return null;
          }
          return {
            id: m.tenant.metadata.id,
            name: m.tenant.name,
            status: TenantStatusType.ACTIVE,
            slug: m.tenant.slug,
            canManage:
              m.role === TenantMemberRole.OWNER ||
              m.role === TenantMemberRole.ADMIN,
          };
        })
        .filter((t): t is OrganizationTenantWithRegion => t !== null) || [],
    [tenantMemberships],
  );

  // showing the first tenant as open, to make clearer that:
  // 1. tenants can expand
  // 2. you can add members to tenants from here
  useEffect(() => {
    const firstVisibleTenantId = visibleTenants[0]?.id;

    if (!firstVisibleTenantId) {
      autoExpandedTenantId.current = null;
      return;
    }

    if (autoExpandedTenantId.current === firstVisibleTenantId) {
      return;
    }

    autoExpandedTenantId.current = firstVisibleTenantId;

    if (!expandedTenantIds.length) {
      setExpandedTenantIds([firstVisibleTenantId]);
    }
  }, [visibleTenants, expandedTenantIds.length]);

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Tenants"
          description="Review the tenants associated with your account."
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
  tenants: OrganizationTenantWithRegion[];
  expandedTenantIds: string[];
  setExpandedTenantIds: (tenantIds: string[]) => void;
  onArchive: (tenant: OrganizationTenantWithRegion) => void;
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
          {tenants.map((tenant, ix) => (
            <div key={tenant.id}>
              <TenantAccordionItem
                key={tenant.id}
                tenant={tenant}
                isExpanded={expandedTenantIds.includes(tenant.id)}
                onArchive={onArchive}
                canManageOrganization={canManageOrganization}
              />
              {ix < tenants.length - 1 && <Separator className="my-4" />}
            </div>
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
  tenant: OrganizationTenantWithRegion;
  isExpanded: boolean;
  onArchive: (tenant: OrganizationTenantWithRegion) => void;
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
  const { isCloudEnabled } = useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();

  const tenantMembers = membersQuery.data?.rows || [];
  const tenantInvites = invitesQuery.data?.rows || [];

  const canManageTenantMembers =
    canManageOrganization ||
    // if both cloud and the control plane are disabled, we're on the OSS and tenant admins / owners can invite members to their tenants
    (!(isCloudEnabled || isControlPlaneEnabled) && Boolean(tenant.canManage));

  return (
    <AccordionItem value={tenant.id} className="overflow-hidden bg-background">
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        <AccordionTrigger className="flex-1 py-1 hover:no-underline [&>svg]:text-muted-foreground">
          <div className="flex min-w-0 items-center gap-2 text-left">
            <p className="min-w-0 truncate font-medium leading-5">
              {tenant.name || tenant.id}
            </p>
            <TenantRegionBadge region={tenant.region} />
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
            {canManageTenantMembers && (
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
              canManage={canManageTenantMembers}
              onMembersChanged={() => membersQuery.refetch()}
            />
          ) : (
            <div className="py-4 text-sm text-muted-foreground">
              No members found.
            </div>
          )}

          {tenantInvites.length > 0 && (
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
  const columns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (member: TenantMember) => (
          <span className="font-medium">{member.user.name}</span>
        ),
      },
      {
        columnLabel: 'Email',
        cellRenderer: (member: TenantMember) => (
          <span className="font-mono text-sm">{member.user.email}</span>
        ),
      },
      {
        columnLabel: 'Role',
        cellRenderer: (member: TenantMember) => (
          <Badge variant="outline">{member.role}</Badge>
        ),
      },
      {
        columnLabel: 'Joined',
        cellRenderer: (member: TenantMember) => (
          <RelativeDate date={member.metadata.createdAt} />
        ),
      },
      ...(canManage
        ? [
            {
              columnLabel: 'Actions',
              cellRenderer: (member: TenantMember) => (
                <TenantMemberActions
                  member={member}
                  tenantId={tenantId}
                  onEditRoleClick={setMemberToEdit}
                  onChangePasswordClick={() => {}}
                  onDeleteSuccess={onMembersChanged}
                />
              ),
            },
          ]
        : []),
    ],
    [canManage, onMembersChanged, tenantId],
  );

  return (
    <>
      <SimpleTable
        columns={columns}
        data={members}
        rowKey={(member) => member.metadata.id}
      />

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
  row: OrganizationTenantWithRegion & { metadata: { id: string } };
  onArchive: (tenant: OrganizationTenantWithRegion) => void;
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
