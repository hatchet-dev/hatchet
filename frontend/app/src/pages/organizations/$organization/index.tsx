import { useParams } from 'react-router-dom';
import { useQuery, useQueries } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import api from '@/lib/api/api';
import { Loading } from '@/components/v1/ui/loading';
import { Button } from '@/components/v1/ui/button';
import {
  PlusIcon,
  BuildingOffice2Icon,
  UserIcon,
  ClipboardIcon,
  CheckIcon,
  KeyIcon,
} from '@heroicons/react/24/outline';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { Badge } from '@/components/v1/ui/badge';
import { useState } from 'react';
import { AddTenantModal } from './components/add-tenant-modal';
import { InviteMemberModal } from './components/invite-member-modal';
import { DeleteMemberModal } from './components/delete-member-modal';
import { CreateTokenModal } from './components/create-token-modal';
import { DeleteTokenModal } from './components/delete-token-modal';
import {
  OrganizationMember,
  ManagementToken,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { EllipsisVerticalIcon, TrashIcon } from '@heroicons/react/24/outline';

// Copy to clipboard component
function CopyableId({
  id,
  className = '',
}: {
  id: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(id);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy: ', err);
    }
  };

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <span className="font-mono text-sm">{id}</span>
      <Button
        variant="ghost"
        size="sm"
        onClick={copyToClipboard}
        className="h-6 w-6 p-0 hover:bg-muted"
      >
        {copied ? (
          <CheckIcon className="h-3 w-3 text-green-600" />
        ) : (
          <ClipboardIcon className="h-3 w-3" />
        )}
      </Button>
    </div>
  );
}

export default function OrganizationPage() {
  const { organization: orgId } = useParams<{ organization: string }>();
  const [showAddTenantModal, setShowAddTenantModal] = useState(false);
  const [showInviteMemberModal, setShowInviteMemberModal] = useState(false);
  const [memberToDelete, setMemberToDelete] =
    useState<OrganizationMember | null>(null);
  const [showCreateTokenModal, setShowCreateTokenModal] = useState(false);
  const [tokenToDelete, setTokenToDelete] = useState<ManagementToken | null>(
    null,
  );

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

  // Fetch detailed tenant information for each tenant - must be called unconditionally
  const tenantQueries = useQueries({
    queries: (organizationQuery.data?.tenants || []).map((tenant) => ({
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

  if (organizationQuery.isLoading) {
    return <Loading />;
  }

  if (organizationQuery.error || !organizationQuery.data) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <h2 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
            Organization not found
          </h2>
          <p className="text-gray-600 dark:text-gray-400 mt-2">
            The organization you're looking for doesn't exist or you don't have
            access to it.
          </p>
        </div>
      </div>
    );
  }

  const organization = organizationQuery.data;

  return (
    <div className="p-6 space-y-6 max-h-full overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-3">
            <BuildingOffice2Icon className="h-8 w-8 text-primary" />
            <div>
              <h1 className="text-3xl font-bold">{organization.name}</h1>
              <p className="text-muted-foreground">
                Manage organization settings
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Organization Details */}
      <Card>
        <CardHeader>
          <CardTitle>Organization Details</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableBody>
              <TableRow>
                <TableCell className="font-medium w-48">
                  Organization ID
                </TableCell>
                <TableCell>
                  <CopyableId id={organization.metadata.id} />
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium w-48">Name</TableCell>
                <TableCell>{organization.name}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium w-48">Slug</TableCell>
                <TableCell className="text-muted-foreground">
                  {organization.slug}
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium w-48">Created</TableCell>
                <TableCell className="text-muted-foreground">
                  {new Date(
                    organization.metadata.createdAt,
                  ).toLocaleDateString()}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Tenants Section */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            Tenants
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAddTenantModal(true)}
            >
              <PlusIcon className="h-4 w-4 mr-2" />
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
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Tenant ID</TableHead>
                  <TableHead>Slug</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {organization.tenants.map((orgTenant) => {
                  const detailedTenant = detailedTenants.find(
                    (t) => t?.metadata.id === orgTenant.id,
                  );
                  return (
                    <TableRow key={orgTenant.id}>
                      <TableCell className="font-medium">
                        {detailedTenant?.name || 'Loading...'}
                      </TableCell>
                      <TableCell>
                        <CopyableId id={orgTenant.id} />
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {detailedTenant?.slug || '-'}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            orgTenant.status === 'active'
                              ? 'default'
                              : 'secondary'
                          }
                        >
                          {orgTenant.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            window.location.href = `/tenants/${orgTenant.id}`;
                          }}
                        >
                          View Tenant
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8">
              <BuildingOffice2Icon className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Tenants Yet</h3>
              <p className="text-muted-foreground mb-4">
                Add your first tenant to get started.
              </p>
              <Button onClick={() => setShowAddTenantModal(true)}>
                <PlusIcon className="h-4 w-4 mr-2" />
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
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowInviteMemberModal(true)}
            >
              <PlusIcon className="h-4 w-4 mr-2" />
              Invite Member
            </Button>
          </CardTitle>
          <CardDescription>
            Members with access to this organization
          </CardDescription>
        </CardHeader>
        <CardContent>
          {organization.members && organization.members.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Member Since</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {organization.members.map((member) => (
                  <TableRow key={member.metadata.id}>
                    <TableCell>
                      <CopyableId id={member.metadata.id} />
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {member.email}
                    </TableCell>
                    <TableCell>
                      <Badge variant="default">{member.memberType}</Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(member.metadata.createdAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                          >
                            <EllipsisVerticalIcon className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => setMemberToDelete(member)}
                            className="text-red-600 focus:text-red-600"
                          >
                            <TrashIcon className="h-4 w-4 mr-2" />
                            Remove Member
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8">
              <UserIcon className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Members Yet</h3>
              <p className="text-muted-foreground mb-4">
                Members will appear here when they join this organization.
              </p>
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
            >
              <PlusIcon className="h-4 w-4 mr-2" />
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
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Token ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {managementTokensQuery.data.rows.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell>
                      <CopyableId id={token.id} />
                    </TableCell>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{token.duration}</Badge>
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => setTokenToDelete(token)}
                      >
                        Delete
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8">
              <KeyIcon className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Management Tokens</h3>
              <p className="text-muted-foreground mb-4">
                Create API tokens to manage this organization programmatically.
              </p>
              <Button onClick={() => setShowCreateTokenModal(true)}>
                <PlusIcon className="h-4 w-4 mr-2" />
                Create Token
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add Tenant Modal */}
      {orgId && organization && (
        <AddTenantModal
          open={showAddTenantModal}
          onOpenChange={setShowAddTenantModal}
          organizationId={orgId}
          organizationName={organization.name}
          onSuccess={() => {
            // Refetch organization data to show new tenant
            organizationQuery.refetch();
          }}
        />
      )}

      {/* Invite Member Modal */}
      {orgId && organization && (
        <InviteMemberModal
          open={showInviteMemberModal}
          onOpenChange={setShowInviteMemberModal}
          organizationId={orgId}
          organizationName={organization.name}
          onSuccess={() => {
            organizationQuery.refetch();
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
    </div>
  );
}
