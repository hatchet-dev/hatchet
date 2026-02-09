import { CancelInviteModal } from './components/cancel-invite-modal';
import { CreateTokenModal } from './components/create-token-modal';
import { DeleteMemberModal } from './components/delete-member-modal';
import { DeleteTenantModal } from './components/delete-tenant-modal';
import { DeleteTokenModal } from './components/delete-token-modal';
import { InviteMemberModal } from './components/invite-member-modal';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useOrganizations } from '@/hooks/use-organizations';
import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import {
  OrganizationMember,
  ManagementToken,
  OrganizationInvite,
  OrganizationInviteStatus,
  TenantStatusType,
  OrganizationTenant,
} from '@/lib/api/generated/cloud/data-contracts';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import { appRoutes } from '@/router';
import {
  PlusIcon,
  BuildingOffice2Icon,
  UserIcon,
  KeyIcon,
  EnvelopeIcon,
  PencilIcon,
  CheckIcon,
  XMarkIcon,
  ArrowRightIcon,
} from '@heroicons/react/24/outline';
import { EllipsisVerticalIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useQuery, useQueries, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { formatDistanceToNow } from 'date-fns';
import { useState } from 'react';

export default function OrganizationPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { handleUpdateOrganization, updateOrganizationLoading } =
    useOrganizations();
  const [showInviteMemberModal, setShowInviteMemberModal] = useState(false);
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
  const [isEditingName, setIsEditingName] = useState(false);
  const [editedName, setEditedName] = useState('');

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
      // Invalidate and refetch the organization query to get updated data
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

      // If the date is in the past, show "expired"
      if (expires < now) {
        return 'expired';
      }

      // Otherwise, show "in X days" format
      return `in ${formatDistanceToNow(expires)}`;
    } catch (error) {
      // Fallback to original date format if parsing fails
      return new Date(expiresDate).toLocaleDateString();
    }
  };

  const organizationQuery = useQuery({
    queryKey: ['organization:get', orgId],
    queryFn: async () => {
      if (!orgId) {
        throw new Error('Organization ID is required');
      }
      const result = await cloudApi.organizationGet(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  // Fetch detailed tenant information for each tenant - only for non-archived tenants
  const tenantQueries = useQueries({
    queries: (organizationQuery.data?.tenants || [])
      .filter((tenant) => tenant.status !== TenantStatusType.ARCHIVED)
      .map((tenant) => ({
        queryKey: ['tenant:get', tenant.id],
        queryFn: async () => {
          const result = await api.tenantGet(tenant.id);
          return result.data;
        },
        enabled: !!tenant.id && !!organizationQuery.data,
      })),
  });

  // Check if all tenant queries are loading
  const tenantsLoading = tenantQueries.some((query) => query.isLoading);

  // Get successful tenant data
  const detailedTenants = tenantQueries
    .filter((query) => query.data)
    .map((query) => query.data);

  // Fetch management tokens for the organization
  const managementTokensQuery = useQuery({
    queryKey: ['management-tokens:list', orgId],
    queryFn: async () => {
      if (!orgId) {
        throw new Error('Organization ID is required');
      }
      const result = await cloudApi.managementTokenList(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  // Fetch organization invites
  const organizationInvitesQuery = useQuery({
    queryKey: ['organization-invites:list', orgId],
    queryFn: async () => {
      if (!orgId) {
        throw new Error('Organization ID is required');
      }
      const result = await cloudApi.organizationInviteList(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  // Get current user to prevent self-deletion
  const { currentUser } = useCurrentUser();

  if (organizationQuery.isLoading) {
    return <Loading />;
  }

  if (organizationQuery.isError) {
    const status = isAxiosError(organizationQuery.error)
      ? organizationQuery.error.response?.status
      : undefined;

    if (status === 404 || status === 403) {
      return (
        <ResourceNotFound
          resource="Organization"
          description="The organization you’re looking for doesn’t exist or you don’t have access to it."
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
    return <Loading />;
  }

  const organization = organizationQuery.data;

  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              {isEditingName ? (
                <>
                  <Input
                    value={editedName}
                    onChange={(e) => setEditedName(e.target.value)}
                    onKeyDown={handleKeyPress}
                    className="h-10 px-3 text-2xl font-bold"
                    autoFocus
                    disabled={updateOrganizationLoading}
                  />
                  <Button
                    size="sm"
                    onClick={handleSaveEdit}
                    disabled={updateOrganizationLoading || !editedName.trim()}
                  >
                    <CheckIcon className="size-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleCancelEdit}
                    disabled={updateOrganizationLoading}
                  >
                    <XMarkIcon className="size-4" />
                  </Button>
                </>
              ) : (
                <>
                  <h1 className="text-2xl font-bold">{organization.name}</h1>
                </>
              )}
              {!isEditingName && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={handleStartEdit}
                  className="h-6 w-6 p-0"
                  disabled={updateOrganizationLoading}
                  style={{ opacity: updateOrganizationLoading ? 0.3 : 1 }}
                >
                  <PencilIcon className="size-3" />
                </Button>
              )}
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              const previousPath = sessionStorage.getItem(
                'orgSettingsPreviousPath',
              );
              if (previousPath) {
                sessionStorage.removeItem('orgSettingsPreviousPath');
                navigate({ to: previousPath, replace: false });
              } else {
                window.history.back();
              }
            }}
            className="h-8 w-8 p-0"
          >
            <XMarkIcon className="size-4" />
          </Button>
        </div>

        {/* Tenants Section */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Tenants
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  navigate({
                    to: appRoutes.onboardingCreateTenantRoute.to,
                    search: { organizationId: organization.metadata.id },
                  });
                }}
                leftIcon={<PlusIcon className="size-4" />}
              >
                Add Tenant
              </Button>
            </CardTitle>
            <CardDescription>Tenants within this organization</CardDescription>
          </CardHeader>
          <CardContent>
            {tenantsLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loading />
              </div>
            ) : organization.tenants && organization.tenants.length > 0 ? (
              <div className="space-y-4">
                {/* Desktop Table View */}
                <div className="hidden md:block">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>ID</TableHead>
                        <TableHead>Slug</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {organization.tenants
                        .filter(
                          (tenant) =>
                            tenant.status !== TenantStatusType.ARCHIVED,
                        )
                        .map((orgTenant) => {
                          const detailedTenant = detailedTenants.find(
                            (t) => t?.metadata.id === orgTenant.id,
                          );
                          return (
                            <TableRow key={orgTenant.id}>
                              <TableCell className="font-medium">
                                {detailedTenant?.name || 'Loading...'}
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-sm">
                                    {orgTenant.id}
                                  </span>
                                  <CopyToClipboard text={orgTenant.id} />
                                </div>
                              </TableCell>
                              <TableCell className="text-muted-foreground">
                                {detailedTenant?.slug || '-'}
                              </TableCell>
                              <TableCell>
                                <DropdownMenu>
                                  <DropdownMenuTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-8 w-8 p-0"
                                    >
                                      <EllipsisVerticalIcon className="size-4" />
                                    </Button>
                                  </DropdownMenuTrigger>
                                  <DropdownMenuContent align="end">
                                    <DropdownMenuItem
                                      onClick={() => {
                                        navigate({
                                          to: appRoutes.tenantRoute.to,
                                          params: { tenant: orgTenant.id },
                                        });
                                      }}
                                    >
                                      <ArrowRightIcon className="mr-2 size-4" />
                                      View Tenant
                                    </DropdownMenuItem>
                                    <DropdownMenuItem
                                      onClick={() =>
                                        setTenantToArchive(orgTenant)
                                      }
                                    >
                                      <TrashIcon className="mr-2 size-4" />
                                      Archive Tenant
                                    </DropdownMenuItem>
                                  </DropdownMenuContent>
                                </DropdownMenu>
                              </TableCell>
                            </TableRow>
                          );
                        })}
                    </TableBody>
                  </Table>
                </div>

                {/* Mobile Card View */}
                <div className="space-y-4 md:hidden">
                  {organization.tenants
                    .filter(
                      (tenant) => tenant.status !== TenantStatusType.ARCHIVED,
                    )
                    .map((orgTenant) => {
                      const detailedTenant = detailedTenants.find(
                        (t) => t?.metadata.id === orgTenant.id,
                      );
                      return (
                        <div
                          key={orgTenant.id}
                          className="space-y-3 rounded-lg border p-4"
                        >
                          <div className="flex items-center justify-between">
                            <h4 className="font-medium">
                              {detailedTenant?.name || 'Loading...'}
                            </h4>
                            <Badge>{orgTenant.status}</Badge>
                          </div>
                          <div className="space-y-2 text-sm">
                            <div>
                              <span className="font-medium text-muted-foreground">
                                Tenant ID:
                              </span>
                              <div className="mt-1 flex items-center gap-2">
                                <span className="font-mono text-sm">
                                  {orgTenant.id}
                                </span>
                                <CopyToClipboard text={orgTenant.id} />
                              </div>
                            </div>
                            <div>
                              <span className="font-medium text-muted-foreground">
                                Slug:
                              </span>
                              <span className="ml-2">
                                {detailedTenant?.slug || '-'}
                              </span>
                            </div>
                          </div>
                          <div className="flex justify-end">
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-8 w-8 p-0"
                                >
                                  <EllipsisVerticalIcon className="size-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={() => {
                                    navigate({
                                      to: appRoutes.tenantRoute.to,
                                      params: { tenant: orgTenant.id },
                                    });
                                  }}
                                >
                                  <ArrowRightIcon className="mr-2 size-4" />
                                  View Tenant
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onClick={() => setTenantToArchive(orgTenant)}
                                >
                                  <TrashIcon className="mr-2 size-4" />
                                  Archive Tenant
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </div>
                        </div>
                      );
                    })}
                </div>
              </div>
            ) : (
              <div className="py-8 text-center">
                <BuildingOffice2Icon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                <h3 className="mb-2 text-lg font-medium">No Tenants Yet</h3>
                <p className="mb-4 text-muted-foreground">
                  Add your first tenant to get started.
                </p>
                <Button
                  onClick={() => {
                    navigate({
                      to: appRoutes.onboardingCreateTenantRoute.to,
                      search: { organizationId: organization.metadata.id },
                    });
                  }}
                  leftIcon={<PlusIcon className="size-4" />}
                >
                  Add Tenant
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Members Section */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Members
            </CardTitle>
            <CardDescription>
              Members with access to this organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            {organization.members && organization.members.length > 0 ? (
              <div className="space-y-4">
                {/* Desktop Table View */}
                <div className="hidden md:block">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>ID</TableHead>
                        <TableHead>Email</TableHead>
                        <TableHead>Role</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {organization.members.map((member) => (
                        <TableRow key={member.metadata.id}>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span className="font-mono text-sm">
                                {member.metadata.id}
                              </span>
                              <CopyToClipboard text={member.metadata.id} />
                            </div>
                          </TableCell>
                          <TableCell className="font-mono text-sm">
                            {member.email}
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">{member.role}</Badge>
                          </TableCell>
                          <TableCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-8 w-8 p-0"
                                >
                                  <EllipsisVerticalIcon className="size-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                {currentUser?.email === member.email ? (
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
                                  <DropdownMenuItem
                                    onClick={() => setMemberToDelete(member)}
                                  >
                                    <TrashIcon className="mr-2 size-4" />
                                    Remove Member
                                  </DropdownMenuItem>
                                )}
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {/* Mobile Card View */}
                <div className="space-y-4 md:hidden">
                  {organization.members.map((member) => (
                    <div
                      key={member.metadata.id}
                      className="space-y-3 rounded-lg border p-4"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm">
                            {member.email}
                          </span>
                          <Badge variant="default">{member.role}</Badge>
                        </div>
                        {currentUser?.email !== member.email && (
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-8 w-8 p-0"
                              >
                                <EllipsisVerticalIcon className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => setMemberToDelete(member)}
                              >
                                <TrashIcon className="mr-2 size-4" />
                                Remove Member
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        )}
                      </div>
                      <div className="space-y-2 text-sm">
                        <div>
                          <span className="font-medium text-muted-foreground">
                            Member ID:
                          </span>
                          <div className="mt-1 flex items-center gap-2">
                            <span className="font-mono text-sm">
                              {member.metadata.id}
                            </span>
                            <CopyToClipboard text={member.metadata.id} />
                          </div>
                        </div>
                        <div>
                          <span className="font-medium text-muted-foreground">
                            Member Since:
                          </span>
                          <span className="ml-2">
                            {new Date(
                              member.metadata.createdAt,
                            ).toLocaleDateString()}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="py-8 text-center">
                <UserIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                <h3 className="mb-2 text-lg font-medium">No Members Yet</h3>
                <p className="mb-4 text-muted-foreground">
                  Members will appear here when they join this organization.
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Organization Invites Section */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Invites
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowInviteMemberModal(true)}
                leftIcon={<PlusIcon className="size-4" />}
              >
                Invite Member
              </Button>
            </CardTitle>
            <CardDescription>
              Pending invitations to join this organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            {organizationInvitesQuery.isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loading />
              </div>
            ) : organizationInvitesQuery.data &&
              organizationInvitesQuery.data.rows &&
              organizationInvitesQuery.data.rows.length > 0 ? (
              <div className="space-y-4">
                {/* Desktop Table View */}
                <div className="hidden md:block">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Email</TableHead>
                        <TableHead>Role</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Expiry</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {organizationInvitesQuery.data.rows
                        .filter(
                          (invite) =>
                            invite.status ===
                              OrganizationInviteStatus.PENDING ||
                            invite.status === OrganizationInviteStatus.EXPIRED,
                        )
                        .map((invite) => (
                          <TableRow key={invite.metadata.id}>
                            <TableCell className="font-mono text-sm">
                              {invite.inviteeEmail}
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline">{invite.role}</Badge>
                            </TableCell>
                            <TableCell>
                              <Badge
                                variant={
                                  invite.status ===
                                  OrganizationInviteStatus.PENDING
                                    ? 'secondary'
                                    : invite.status ===
                                        OrganizationInviteStatus.ACCEPTED
                                      ? 'default'
                                      : 'destructive'
                                }
                              >
                                {invite.status}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              {formatExpirationDate(invite.expires)}
                            </TableCell>
                            <TableCell>
                              {invite.status ===
                                OrganizationInviteStatus.PENDING && (
                                <DropdownMenu>
                                  <DropdownMenuTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-8 w-8 p-0"
                                    >
                                      <EllipsisVerticalIcon className="size-4" />
                                    </Button>
                                  </DropdownMenuTrigger>
                                  <DropdownMenuContent align="end">
                                    <DropdownMenuItem
                                      onClick={() => setInviteToCancel(invite)}
                                    >
                                      <TrashIcon className="mr-2 size-4" />
                                      Cancel Invitation
                                    </DropdownMenuItem>
                                  </DropdownMenuContent>
                                </DropdownMenu>
                              )}
                            </TableCell>
                          </TableRow>
                        ))}
                    </TableBody>
                  </Table>
                </div>

                {/* Mobile Card View */}
                <div className="space-y-4 md:hidden">
                  {organizationInvitesQuery.data.rows.map((invite) => (
                    <div
                      key={invite.metadata.id}
                      className="space-y-3 rounded-lg border p-4"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm">
                            {invite.inviteeEmail}
                          </span>
                          <Badge variant="outline">{invite.role}</Badge>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge
                            variant={
                              invite.status === OrganizationInviteStatus.PENDING
                                ? 'secondary'
                                : invite.status ===
                                    OrganizationInviteStatus.ACCEPTED
                                  ? 'default'
                                  : 'destructive'
                            }
                          >
                            {invite.status}
                          </Badge>
                          {invite.status ===
                            OrganizationInviteStatus.PENDING && (
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-8 w-8 p-0"
                                >
                                  <EllipsisVerticalIcon className="size-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={() => setInviteToCancel(invite)}
                                >
                                  <TrashIcon className="mr-2 size-4" />
                                  Cancel Invitation
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          )}
                        </div>
                      </div>
                      <div className="space-y-2 text-sm">
                        <div>
                          <span className="font-medium text-muted-foreground">
                            Invite ID:
                          </span>
                          <div className="mt-1 flex items-center gap-2">
                            <span className="font-mono text-sm">
                              {invite.metadata.id}
                            </span>
                            <CopyToClipboard text={invite.metadata.id} />
                          </div>
                        </div>
                        <div>
                          <span className="font-medium text-muted-foreground">
                            Invited By:
                          </span>
                          <span className="ml-2">{invite.inviterEmail}</span>
                        </div>
                        <div>
                          <span className="font-medium text-muted-foreground">
                            Expires:
                          </span>
                          <span className="ml-2">
                            {formatExpirationDate(invite.expires)}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="py-8 text-center">
                <EnvelopeIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                <h3 className="mb-2 text-lg font-medium">No Pending Invites</h3>
                <p className="mb-4 text-muted-foreground">
                  Invite members to join this organization.
                </p>
                <Button
                  onClick={() => setShowInviteMemberModal(true)}
                  leftIcon={<PlusIcon className="size-4" />}
                >
                  Invite Member
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Management Tokens Section */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              Management Tokens
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowCreateTokenModal(true)}
                leftIcon={<PlusIcon className="size-4" />}
              >
                Create Token
              </Button>
            </CardTitle>
            <CardDescription>
              API tokens for managing this organization
            </CardDescription>
          </CardHeader>
          <CardContent>
            {managementTokensQuery.isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loading />
              </div>
            ) : managementTokensQuery.data &&
              managementTokensQuery.data.rows &&
              managementTokensQuery.data.rows.length > 0 ? (
              <div className="space-y-4">
                {/* Desktop Table View */}
                <div className="hidden md:block">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>ID</TableHead>
                        <TableHead>Name</TableHead>
                        <TableHead>Expiry</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {managementTokensQuery.data.rows.map((token) => (
                        <TableRow key={token.id}>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span className="font-mono text-sm">
                                {token.id}
                              </span>
                              <CopyToClipboard text={token.id} />
                            </div>
                          </TableCell>
                          <TableCell className="font-medium">
                            {token.name}
                          </TableCell>
                          <TableCell>
                            {formatExpirationDate(token.expiresAt)}
                          </TableCell>
                          <TableCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-8 w-8 p-0"
                                >
                                  <EllipsisVerticalIcon className="size-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={() => setTokenToDelete(token)}
                                >
                                  <TrashIcon className="mr-2 size-4" />
                                  Delete
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {/* Mobile Card View */}
                <div className="space-y-4 md:hidden">
                  {managementTokensQuery.data.rows.map((token) => (
                    <div
                      key={token.id}
                      className="space-y-3 rounded-lg border p-4"
                    >
                      <div className="flex items-center justify-between">
                        <h4 className="font-medium">{token.name}</h4>
                        <div className="flex items-center gap-2">
                          <Badge variant="outline">
                            {formatExpirationDate(token.expiresAt)}
                          </Badge>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-8 w-8 p-0"
                              >
                                <EllipsisVerticalIcon className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => setTokenToDelete(token)}
                              >
                                <TrashIcon className="mr-2 size-4" />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </div>
                      <div className="space-y-2 text-sm">
                        <div>
                          <span className="font-medium text-muted-foreground">
                            Token ID:
                          </span>
                          <div className="mt-1 flex items-center gap-2">
                            <span className="font-mono text-sm">
                              {token.id}
                            </span>
                            <CopyToClipboard text={token.id} />
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="py-8 text-center">
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
          </CardContent>
        </Card>

        {/* Invite Member Modal */}
        {orgId && organization && (
          <InviteMemberModal
            open={showInviteMemberModal}
            onOpenChange={setShowInviteMemberModal}
            organizationId={orgId}
            organizationName={organization.name}
            onSuccess={() => {
              organizationQuery.refetch();
              organizationInvitesQuery.refetch();
            }}
          />
        )}

        {/* Delete Member Modal */}
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

        {/* Create Token Modal */}
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

        {/* Delete Token Modal */}
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

        {/* Cancel Invite Modal */}
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

        {/* Archive Tenant Modal */}
        {(() => {
          const foundTenant = tenantToArchive
            ? detailedTenants.find((t) => t?.metadata.id === tenantToArchive.id)
            : undefined;
          return (
            tenantToArchive &&
            organization &&
            foundTenant && (
              <DeleteTenantModal
                open={!!tenantToArchive}
                onOpenChange={(open) => !open && setTenantToArchive(null)}
                tenant={tenantToArchive}
                tenantName={foundTenant.name}
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
    </div>
  );
}
