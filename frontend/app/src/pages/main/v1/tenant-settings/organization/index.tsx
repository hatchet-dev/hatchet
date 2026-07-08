import {
  formatMemberRole,
  MemberEmail,
  RoleBadge,
} from '../components/member-primitives';
import { SettingsPageHeader } from '../components/settings-page-header';
import { SettingRow } from '../components/settings-row';
import { usePylon } from '@/components/support-chat';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { TenantRegionBadge } from '@/components/v1/molecules/nav-bar/tenant-region-badge';
import {
  SearchBarWithFilters,
  type SearchSuggestion,
} from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
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
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { Switch } from '@/components/v1/ui/switch';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentUser } from '@/hooks/use-current-user';
import {
  MAX_INACTIVITY_TIMEOUT_MS,
  useOrganizations,
} from '@/hooks/use-organizations';
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
  OrganizationInviteTenant,
  OrganizationTenant as ControlPlaneOrganizationTenant,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { OFFICE_HOURS_URL } from '@/lib/external-links';
import { globalEmitter } from '@/lib/global-emitter';
import {
  formatShardDeploymentKey,
  shardDeploymentKey,
} from '@/lib/shard-deployment-key';
import { parseDuration, msToDurationString } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta.ts';
import { AuditLogSettings } from '@/pages/main/v1/tenant-settings/organization/audit-log-settings';
import CreateSSOPage from '@/pages/main/v1/tenant-settings/organization/components/sso-setup.tsx';
import {
  TagBadge,
  TagList,
} from '@/pages/main/v1/tenant-settings/organization/components/tag-badge';
import { UserGroupsTab } from '@/pages/main/v1/tenant-settings/organization/components/user-groups-tab';
import { CancelInviteModal } from '@/pages/organizations/$organization/components/cancel-invite-modal';
import { CreateTokenModal } from '@/pages/organizations/$organization/components/create-token-modal';
import { DeleteMemberModal } from '@/pages/organizations/$organization/components/delete-member-modal';
import { DeleteTenantModal } from '@/pages/organizations/$organization/components/delete-tenant-modal';
import { DeleteTokenModal } from '@/pages/organizations/$organization/components/delete-token-modal';
import { EditTenantTagsModal } from '@/pages/organizations/$organization/components/edit-tenant-tags-modal';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import {
  PlusIcon,
  ArrowRightIcon,
  CheckIcon,
  EllipsisVerticalIcon,
  ExclamationTriangleIcon,
  PencilSquareIcon,
  TrashIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { formatDistanceToNow } from 'date-fns';
import { useMemo, useState } from 'react';

// FIXME: remove this once we migrate everyone to the control plane
export type OrganizationTenantWithRegion = OrganizationTenant & {
  region?: ControlPlaneOrganizationTenant['region'];
};

const TERRAFORM_PROVIDER_DOCS_URL =
  'https://registry.terraform.io/providers/hatchet-dev/hatchet/latest/docs';

const noopAutocomplete = () => ({ suggestions: [] as SearchSuggestion[] });
const noopApplySuggestion = (query: string) => query;
const emptyAutocompleteContext = {};

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

// The cloud client's OrganizationInvite lacks the control-plane-only `tenants`
// field. `tenants` is absent (not `[]`) when there are no grants or the server
// is older — never assume it exists.
type OrganizationInviteWithTenants = OrganizationInvite & {
  tenants?: OrganizationInviteTenant[];
};

export type OrganizationSettingsSection =
  | 'general'
  | 'tenants'
  | 'team'
  | 'tokens'
  | 'regions'
  | 'sso'
  | 'audit-log';

const SECTION_HEADERS: Record<
  OrganizationSettingsSection,
  { title: string; description: string }
> = {
  general: {
    title: 'General',
    description: 'Update the organization name and inactivity timeout.',
  },
  tenants: {
    title: 'Tenants',
    description: 'Manage the tenants associated with this organization.',
  },
  team: {
    title: 'Team',
    description:
      'Manage the members, invites, and user groups for this organization.',
  },
  tokens: {
    title: 'Management Tokens',
    description:
      'Organization-scoped API tokens for automating administrative tasks — like creating tenants and managing members — via the Hatchet API.',
  },
  regions: {
    title: 'Available Regions',
    description:
      'Review the regions where new tenants can be deployed for this organization.',
  },
  sso: {
    title: 'SSO',
    description: 'Configure single sign-on for this organization.',
  },
  'audit-log': {
    title: 'Audit Log',
    description: 'Review administrative actions taken in this organization.',
  },
};

function InviteTenantBadges({
  tenants,
}: {
  tenants?: OrganizationInviteTenant[];
}) {
  if (!tenants || tenants.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }

  const visible = tenants.slice(0, 3);
  const overflow = tenants.slice(3);

  return (
    <div className="flex flex-wrap items-center gap-1">
      {visible.map((tenant) => (
        <Badge key={tenant.tenantId} variant="outline">
          {tenant.tenantName} · {formatMemberRole(tenant.tenantRole)}
        </Badge>
      ))}
      {overflow.length > 0 && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="outline">+{overflow.length} more</Badge>
            </TooltipTrigger>
            <TooltipContent>
              <div className="flex flex-col gap-1">
                {overflow.map((tenant) => (
                  <span key={tenant.tenantId}>
                    {tenant.tenantName} · {formatMemberRole(tenant.tenantRole)}
                  </span>
                ))}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  );
}

function SectionUnavailable({ description }: { description?: string }) {
  return (
    <div className="py-12">
      <EmptyState
        title="You do not have access to this page"
        description={
          description ?? 'Ask an organization owner to grant you access.'
        }
      />
    </div>
  );
}

export function CloudOrganizationSettings({
  orgId,
  section = 'general',
}: {
  orgId: string;
  section?: OrganizationSettingsSection;
}) {
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
  const canManageSso = isOrganizationOwner && schemes.includes('sso');
  const [memberToDelete, setMemberToDelete] =
    useState<OrganizationMember | null>(null);
  const [tenantToEditTags, setTenantToEditTags] =
    useState<OrganizationTenantWithRegion | null>(null);
  const [showCreateTokenModal, setShowCreateTokenModal] = useState(false);
  const [tokenToDelete, setTokenToDelete] = useState<ManagementToken | null>(
    null,
  );
  const [inviteToCancel, setInviteToCancel] =
    useState<OrganizationInvite | null>(null);
  const [tenantToArchive, setTenantToArchive] =
    useState<OrganizationTenantWithRegion | null>(null);
  const [editedName, setEditedName] = useState('');
  const [isEditingName, setIsEditingName] = useState(false);
  const [editedTimeout, setEditedTimeout] = useState('');
  const [isEditingTimeout, setIsEditingTimeout] = useState(false);
  const [newSsoDomain, setNewSsoDomain] = useState('');
  const [isAddingSsoDomain, setIsAddingSsoDomain] = useState(false);
  const [ssoIsConfigured, setSsoIsConfigured] = useState(false);

  const organizationEntitlementsQuery = useQuery({
    ...orgApi.organizationEntitlementsGetQuery(orgId!),
    enabled: !!orgId && canManageSso,
  });
  const canUseSso = organizationEntitlementsQuery.data?.canSSO === true;

  const organizationSsoDomainGetQuery = useQuery({
    ...orgApi.organizationSsoDomainGetQuery(orgId),
    enabled: !!orgId && canUseSso,
  });

  const organizationSsoConfigGetQuery = useQuery({
    ...orgApi.organizationSsoConfigGetQuery(orgId),
    enabled: !!orgId && canUseSso,
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

  const allTenantTags = useMemo(() => {
    const tagSet = new Set<string>();
    for (const tenant of visibleTenants) {
      const tags = (tenant as unknown as { tags?: string[] }).tags;
      tags?.forEach((t) => tagSet.add(t));
    }
    return Array.from(tagSet).sort();
  }, [visibleTenants]);

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

  const editedTimeoutExceedsMax =
    parsedEditedTimeout !== null &&
    parsedEditedTimeout > MAX_INACTIVITY_TIMEOUT_MS;

  const handleSaveTimeout = () => {
    if (!orgId || parsedEditedTimeout === null || editedTimeoutExceedsMax) {
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
        <MemberEmail email={row.email} />
      ),
    },
    {
      columnLabel: 'Role',
      cellRenderer: (row: OrganizationMember) => <RoleBadge role={row.role} />,
    },
    ...(isOrganizationOwner
      ? [
          {
            columnLabel: 'Actions',
            cellRenderer: (row: OrganizationMember) => (
              <MemberActions
                row={row}
                organizationId={orgId}
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
      cellRenderer: (row: OrganizationInviteWithTenants) => (
        <MemberEmail email={row.inviteeEmail} />
      ),
    },
    {
      columnLabel: 'Role',
      cellRenderer: (row: OrganizationInviteWithTenants) => (
        <RoleBadge role={row.role} />
      ),
    },
    {
      columnLabel: 'Tenant Access',
      cellRenderer: (row: OrganizationInviteWithTenants) => (
        <InviteTenantBadges tenants={row.tenants} />
      ),
    },
    {
      columnLabel: 'Status',
      cellRenderer: (row: OrganizationInviteWithTenants) => (
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
      cellRenderer: (row: OrganizationInviteWithTenants) => (
        <span>{formatExpiry(row.expires)}</span>
      ),
    },
    ...(isOrganizationOwner
      ? [
          {
            columnLabel: 'Actions',
            cellRenderer: (row: OrganizationInviteWithTenants) =>
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
        <span className="font-mono text-sm">
          {formatShardDeploymentKey(shardDeploymentKey(row)) ?? row.region}
        </span>
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
      columnLabel: 'Tags',
      cellRenderer: (row: ManagementToken) => {
        const rowTags = (row as unknown as { tags?: string[] }).tags;
        return rowTags && rowTags.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {rowTags.map((tag) => (
              <TagBadge key={tag} tag={tag} />
            ))}
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        );
      },
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
          title={SECTION_HEADERS[section].title}
          description={SECTION_HEADERS[section].description}
        />

        {section === 'general' && !isOrganizationOwner && (
          <SectionUnavailable description="Only organization owners can manage these settings." />
        )}

        {section === 'general' && isOrganizationOwner && (
          <div className="divide-y divide-border">
            <SettingRow
              label="Organization Name"
              description="The display name for this organization, shown across the dashboard."
            >
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
                    className="w-[220px]"
                    disabled={updateOrganizationLoading}
                    aria-label="Organization name"
                    autoFocus
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleCancelEditingName}
                    disabled={updateOrganizationLoading}
                    hoverText="Cancel editing"
                    aria-label="Cancel editing"
                    className="shrink-0"
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
                    aria-label="Save organization name"
                    className="shrink-0"
                  >
                    {updateOrganizationLoading ? (
                      <Spinner />
                    ) : (
                      <CheckIcon className="size-4" />
                    )}
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="max-w-[220px] truncate text-sm">
                    {organizationName}
                  </span>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={handleStartEditingName}
                    hoverText="Edit organization name"
                    aria-label="Edit organization name"
                    className="shrink-0"
                  >
                    <PencilSquareIcon className="size-4" />
                  </Button>
                </div>
              )}
            </SettingRow>

            {isControlPlaneEnabled && (
              <SettingRow
                label="Inactivity Timeout"
                description="Automatically sign out members of this organization after this period of inactivity. Maximum 14 days."
              >
                {isEditingTimeout ? (
                  <div className="flex flex-col items-end gap-1.5">
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
                        className="w-[220px]"
                        placeholder="e.g. 30m, 1h, 1h30m, -1 to disable"
                        disabled={updateOrganizationLoading}
                        autoFocus
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={handleCancelEditingTimeout}
                        disabled={updateOrganizationLoading}
                        hoverText="Cancel editing"
                        aria-label="Cancel editing"
                        className="shrink-0"
                      >
                        <XMarkIcon className="size-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={handleSaveTimeout}
                        disabled={
                          updateOrganizationLoading ||
                          parsedEditedTimeout === null ||
                          editedTimeoutExceedsMax
                        }
                        hoverText="Save inactivity timeout"
                        aria-label="Save inactivity timeout"
                        className="shrink-0"
                      >
                        {updateOrganizationLoading ? (
                          <Spinner />
                        ) : (
                          <CheckIcon className="size-4" />
                        )}
                      </Button>
                    </div>
                    {editedTimeout.trim() !== '' && (
                      <p
                        className={`text-xs ${
                          parsedEditedTimeout === null ||
                          editedTimeoutExceedsMax
                            ? 'text-destructive'
                            : 'text-muted-foreground'
                        }`}
                      >
                        {parsedEditedTimeout === null
                          ? 'Invalid format — try 30m, 1h, 1h30m, 100ms'
                          : editedTimeoutExceedsMax
                            ? 'Inactivity timeout cannot exceed 14 days'
                            : `→ ${formatTimeoutMs(parsedEditedTimeout)}`}
                      </p>
                    )}
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <span className="max-w-[220px] truncate text-sm">
                      {formatTimeoutMs(currentInactivityTimeoutMs)}
                    </span>
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={handleStartEditingTimeout}
                      hoverText="Edit inactivity timeout"
                      aria-label="Edit inactivity timeout"
                      className="shrink-0"
                    >
                      <PencilSquareIcon className="size-4" />
                    </Button>
                  </div>
                )}
              </SettingRow>
            )}
          </div>
        )}

        <div className="mt-2">
          {section === 'tenants' && (
            <TenantsSection
              tenants={visibleTenants}
              onArchive={setTenantToArchive}
              onEditTags={
                isControlPlaneEnabled && isOrganizationOwner
                  ? setTenantToEditTags
                  : undefined
              }
              defaultOrganizationId={orgId}
              canManageOrganization={isOrganizationOwner}
            />
          )}

          {section === 'team' &&
            (organizationQuery.error instanceof AxiosError &&
            organizationQuery.error.response?.status === 403 ? (
              <SectionUnavailable description="You must be an organization owner to view members." />
            ) : (
              <div className="space-y-8">
                <div className="space-y-4">
                  <div>
                    <h3 className="text-base font-semibold">Members</h3>
                    <p className="mt-1 text-sm text-muted-foreground">
                      People with access to this organization.
                    </p>
                  </div>
                  <Separator />
                  {isOrganizationOwner && (
                    <div className="flex justify-end">
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
                    <div className="py-8">
                      <EmptyState
                        title="No members"
                        description="Invite teammates to give them access to this organization."
                      />
                    </div>
                  )}
                </div>

                {pendingInvites && pendingInvites.length > 0 && (
                  <div className="space-y-4">
                    <div>
                      <h3 className="text-base font-semibold">
                        Pending Invites
                      </h3>
                      <p className="mt-1 text-sm text-muted-foreground">
                        Invitations that have not been accepted yet.
                      </p>
                    </div>
                    <Separator />
                    <SimpleTable
                      data={pendingInvites}
                      columns={inviteColumns}
                      rowKey={(row) => row.metadata.id}
                    />
                  </div>
                )}

                {isOrganizationOwner && isControlPlaneEnabled && (
                  <div className="space-y-4">
                    <div>
                      <h3 className="text-base font-semibold">User Groups</h3>
                      <p className="mt-1 text-sm text-muted-foreground">
                        Manage user groups for this organization.
                      </p>
                    </div>
                    <Separator />
                    <UserGroupsTab
                      organizationId={orgId}
                      allOrgMembers={organization?.members ?? []}
                      allTenantTags={allTenantTags}
                    />
                  </div>
                )}
              </div>
            ))}

          {section === 'regions' &&
            (!(isOrganizationOwner && isControlPlaneEnabled) ? (
              <SectionUnavailable />
            ) : organizationAvailableShardsQuery.isLoading ? (
              <div className="flex justify-center py-12">
                <Loading />
              </div>
            ) : organizationAvailableShardsQuery.error instanceof AxiosError &&
              organizationAvailableShardsQuery.error.response?.status ===
                403 ? (
              <SectionUnavailable description="You must be an organization owner to view available regions." />
            ) : organizationAvailableShardsQuery.error ? (
              <div className="py-8">
                <EmptyState
                  title="Failed to load available regions"
                  description="Something went wrong fetching the regions for this organization. Try refreshing the page."
                />
              </div>
            ) : (
              <div className="space-y-4">
                {organizationAvailableShardsQuery.data?.rows &&
                organizationAvailableShardsQuery.data.rows.length > 0 ? (
                  <SimpleTable
                    data={organizationAvailableShardsQuery.data.rows}
                    columns={availableShardColumns}
                    rowKey={(row) =>
                      `${row.shardClass}:${row.provider}:${row.region}:${row.shardName ?? ''}`
                    }
                  />
                ) : (
                  <div className="py-8">
                    <EmptyState
                      title="No deployment regions"
                      description="No regions are currently available for deploying new tenants in this organization."
                    />
                  </div>
                )}
                <div className="space-y-2 text-sm text-muted-foreground">
                  <p>
                    Need to configure which regions are available for a tenant,
                    or looking for a new region?{' '}
                    {pylon.enabled ? (
                      <>
                        <Button
                          type="button"
                          variant="link"
                          className="h-auto p-0 text-sm font-normal"
                          onClick={() => pylon.show()}
                        >
                          Chat with us
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
              </div>
            ))}

          {section === 'tokens' &&
            (managementTokensQuery.error instanceof AxiosError &&
            managementTokensQuery.error.response?.status === 403 ? (
              <SectionUnavailable description="You must be an organization owner to view management tokens." />
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
                  <div className="py-8">
                    <EmptyState
                      title="No management tokens"
                      description="Management tokens are used to manage tenants and members programmatically. We recommend using the Hatchet Terraform provider for this."
                      docPage={{ href: TERRAFORM_PROVIDER_DOCS_URL }}
                      docLabel="Hatchet Terraform provider docs"
                    />
                  </div>
                )}
              </>
            ))}

          {section === 'sso' &&
            (!canManageSso ? (
              <SectionUnavailable />
            ) : organizationEntitlementsQuery.isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loading />
              </div>
            ) : canUseSso ? (
              <div className="space-y-6">
                <CreateSSOPage
                  orgId={orgId}
                  onConfigLoaded={setSsoIsConfigured}
                />
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
                {/* SSO Domains */}
                <div className="space-y-4">
                  <div>
                    <h3 className="text-base font-semibold">SSO Domains</h3>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Domains associated with your organization for SSO login.
                      Members signing in with a verified domain will be
                      automatically directed to your identity provider.
                    </p>
                  </div>
                  {ssoIsConfigured &&
                    !organizationSsoDomainGetQuery.isLoading &&
                    (!organizationSsoDomainGetQuery.data ||
                      organizationSsoDomainGetQuery.data.length === 0) && (
                      <div className="flex items-start gap-3 rounded-md border border-yellow-500/50 bg-yellow-500/10 px-4 py-3 text-sm">
                        <ExclamationTriangleIcon className="mt-0.5 h-4 w-4 shrink-0 text-yellow-500" />
                        <div>
                          <p className="font-medium text-yellow-600 dark:text-yellow-400">
                            SSO is configured but no domains are set up.
                          </p>
                          <p className="mt-0.5 text-muted-foreground">
                            Without a verified domain, members will not be
                            automatically redirected to your identity provider.
                            Add a domain below to complete your SSO setup.
                          </p>
                        </div>
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
                  ) : null}
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
              </div>
            ) : (
              <div className="py-12">
                <EmptyState
                  title="SSO is not enabled"
                  description="Single sign-on is not enabled for this organization. Contact us to get access."
                  links={[
                    {
                      href: OFFICE_HOURS_URL,
                      label: 'Schedule office hours',
                      external: true,
                    },
                  ]}
                />
              </div>
            ))}

          {section === 'audit-log' &&
            (isControlPlaneEnabled ? (
              <AuditLogSettings orgId={orgId} />
            ) : (
              <SectionUnavailable />
            ))}
        </div>
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
          allTenantTags={allTenantTags}
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

      {isOrganizationOwner && isControlPlaneEnabled && tenantToEditTags && (
        <EditTenantTagsModal
          open={!!tenantToEditTags}
          onOpenChange={(open) => !open && setTenantToEditTags(null)}
          organizationId={orgId}
          tenantId={tenantToEditTags.id}
          tenantName={tenantToEditTags.name || tenantToEditTags.id}
          initialTags={(tenantToEditTags as { tags?: string[] }).tags ?? []}
          allTenantTags={allTenantTags}
          onSuccess={() =>
            queryClient.invalidateQueries({
              queryKey: ['organization:get', orgId],
            })
          }
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
          };
        })
        .filter((t): t is OrganizationTenantWithRegion => t !== null) || [],
    [tenantMemberships],
  );

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Tenants"
          description="Review the tenants associated with your account."
        />

        <TenantsSection
          tenants={visibleTenants}
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
  onArchive,
  onEditTags,
  defaultOrganizationId,
  canManageOrganization,
}: {
  tenants: OrganizationTenantWithRegion[];
  onArchive: (tenant: OrganizationTenantWithRegion) => void;
  onEditTags?: (tenant: OrganizationTenantWithRegion) => void;
  defaultOrganizationId?: string;
  canManageOrganization: boolean;
}) {
  const navigate = useNavigate();
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [search, setSearch] = useState('');

  const allTags = useMemo(() => {
    const tagSet = new Set<string>();
    for (const tenant of tenants) {
      const tags = (tenant as unknown as { tags?: string[] }).tags;
      tags?.forEach((t) => tagSet.add(t));
    }
    return Array.from(tagSet).sort();
  }, [tenants]);

  const filteredTenants = useMemo(() => {
    const term = search.trim().toLowerCase();

    return tenants.filter((tenant) => {
      if (term) {
        const haystack = [tenant.name, tenant.slug, tenant.id]
          .filter(Boolean)
          .join(' ')
          .toLowerCase();

        if (!haystack.includes(term)) {
          return false;
        }
      }

      if (selectedTags.length > 0) {
        const tags = (tenant as unknown as { tags?: string[] }).tags ?? [];
        return selectedTags.every((t) => tags.includes(t));
      }

      return true;
    });
  }, [tenants, selectedTags, search]);

  const isFiltered = search.trim() !== '' || selectedTags.length > 0;

  const toggleTag = (tag: string) =>
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );

  const tenantColumns = [
    {
      columnLabel: 'Name',
      cellRenderer: (tenant: OrganizationTenantWithRegion) => (
        <p className="min-w-0 truncate font-medium">
          {tenant.name ?? tenant.slug ?? tenant.id}
        </p>
      ),
    },
    {
      columnLabel: 'ID',
      cellRenderer: (tenant: OrganizationTenantWithRegion) => (
        <div className="flex items-center gap-2">
          <span className="max-w-[10rem] truncate font-mono text-xs text-muted-foreground">
            {tenant.id}
          </span>
          <CopyToClipboard text={tenant.id} />
        </div>
      ),
    },
    {
      columnLabel: 'Tags',
      cellRenderer: (tenant: OrganizationTenantWithRegion) => {
        const tags = (tenant as unknown as { tags?: string[] }).tags ?? [];
        return tags.length > 0 ? (
          <TagList tags={tags} />
        ) : (
          <span className="text-muted-foreground">—</span>
        );
      },
    },
    {
      columnLabel: 'Status',
      cellRenderer: (tenant: OrganizationTenantWithRegion) => (
        <Badge variant="outline">{tenant.status}</Badge>
      ),
    },
    {
      columnLabel: 'Region',
      cellRenderer: (tenant: OrganizationTenantWithRegion) =>
        tenant.region ? (
          <TenantRegionBadge region={tenant.region} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      columnLabel: 'Members',
      cellRenderer: (tenant: OrganizationTenantWithRegion) => (
        <Button
          variant="outline"
          size="sm"
          onClick={() =>
            navigate({
              to: appRoutes.tenantSettingsMembersRoute.to,
              params: { tenant: tenant.id },
            })
          }
        >
          Manage members
        </Button>
      ),
    },
    {
      columnLabel: 'Actions',
      cellRenderer: (tenant: OrganizationTenantWithRegion) => (
        <TenantActions
          row={{ ...tenant, metadata: { id: tenant.id } }}
          onArchive={onArchive}
          onEditTags={onEditTags}
          canManageOrganization={canManageOrganization}
        />
      ),
    },
  ];

  return (
    <>
      <div className="mb-4 flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <SearchBarWithFilters
            value={search}
            onChange={setSearch}
            onSubmit={setSearch}
            getAutocomplete={noopAutocomplete}
            applySuggestion={noopApplySuggestion}
            autocompleteContext={emptyAutocompleteContext}
            placeholder="Search tenants by name, slug, or ID..."
          />
        </div>
        {canManageOrganization && (
          <div className="shrink-0">
            <Button
              onClick={() =>
                globalEmitter.emit('create-new-tenant', {
                  defaultOrganizationId,
                  allTenantTags: allTags,
                })
              }
            >
              Add Tenant
            </Button>
          </div>
        )}
      </div>
      {allTags.length > 0 && (
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">Filter by tag:</span>
          {allTags.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={() => toggleTag(tag)}
              className="focus:outline-none"
            >
              <Badge
                variant={selectedTags.includes(tag) ? 'default' : 'outline'}
                className="cursor-pointer text-xs"
              >
                {tag}
              </Badge>
            </button>
          ))}
          {selectedTags.length > 0 && (
            <button
              type="button"
              onClick={() => setSelectedTags([])}
              className="text-xs text-muted-foreground hover:text-foreground focus:outline-none"
            >
              <XMarkIcon className="inline size-3" /> Clear
            </button>
          )}
        </div>
      )}
      {filteredTenants.length ? (
        <SimpleTable
          columns={tenantColumns}
          data={filteredTenants}
          rowKey={(tenant) => tenant.id}
        />
      ) : (
        <div className="py-8">
          <EmptyState
            title="No tenants"
            description={
              isFiltered
                ? 'No tenants match your search or filters.'
                : 'Add a tenant to this organization to get started.'
            }
            filterHint={
              isFiltered
                ? 'Try changing your search or clearing the tag filters.'
                : undefined
            }
          />
        </div>
      )}
    </>
  );
}

function TenantActions({
  row,
  onArchive,
  onEditTags,
  canManageOrganization,
}: {
  row: OrganizationTenantWithRegion & { metadata: { id: string } };
  onArchive: (tenant: OrganizationTenantWithRegion) => void;
  onEditTags?: (tenant: OrganizationTenantWithRegion) => void;
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
        {canManageOrganization && onEditTags && (
          <DropdownMenuItem onClick={() => onEditTags(row)}>
            <PencilSquareIcon className="mr-2 size-4" />
            Edit Tags
          </DropdownMenuItem>
        )}
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
  organizationId,
  currentUserEmail,
  onDelete,
}: {
  row: OrganizationMember;
  organizationId: string;
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
        <DropdownMenuItem
          onClick={() =>
            globalEmitter.emit('create-tenant-invite', {
              organizationId,
              defaultEmail: row.email,
            })
          }
        >
          <PlusIcon className="mr-2 size-4" />
          Add to tenant
        </DropdownMenuItem>
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
