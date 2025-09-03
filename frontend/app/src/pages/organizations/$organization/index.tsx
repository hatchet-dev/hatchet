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

export default function OrganizationPage() {
  const { organization: orgId } = useParams<{ organization: string }>();

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
    <div className="p-6 space-y-6">
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
                <TableCell className="font-mono text-sm">
                  {organization.metadata.id}
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
              onClick={() => {
                window.location.href = `/organizations/${orgId}/add-tenant`;
              }}
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
                      <TableCell className="font-mono text-sm">
                        {orgTenant.id}
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
              <Button
                onClick={() => {
                  window.location.href = `/organizations/${orgId}/add-tenant`;
                }}
              >
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
          <CardTitle>Members</CardTitle>
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
                </TableRow>
              </TableHeader>
              <TableBody>
                {organization.members.map((member) => (
                  <TableRow key={member.metadata.id}>
                    <TableCell className="font-mono text-sm">
                      {member.metadata.id}
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
    </div>
  );
}
