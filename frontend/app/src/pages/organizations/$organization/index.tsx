import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { Loading } from '@/components/v1/ui/loading';
import { Button } from '@/components/v1/ui/button';
import {
  PlusIcon,
  Cog6ToothIcon,
  BuildingOffice2Icon,
} from '@heroicons/react/24/outline';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';

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
                Manage organization settings and tenants
              </p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
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
          <Button variant="outline" size="sm">
            <Cog6ToothIcon className="h-4 w-4 mr-2" />
            Settings
          </Button>
        </div>
      </div>

      {/* Tenants Section */}
      <Card>
        <CardHeader>
          <CardTitle>Tenants</CardTitle>
          <CardDescription>Tenants within this organization</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8">
            <BuildingOffice2Icon className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Tenant Management</h3>
            <p className="text-muted-foreground mb-4">
              Add and manage tenants for this organization.
            </p>
            <Button
              onClick={() => {
                window.location.replace(
                  `${window.location.protocol}//${window.location.host}/organizations/${orgId}/add-tenant`,
                );
              }}
            >
              <PlusIcon className="h-4 w-4 mr-2" />
              Add Tenant
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Organization Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Organization Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Organization ID</label>
              <p className="text-sm text-muted-foreground font-mono">
                {organization.metadata.id}
              </p>
            </div>
            <div>
              <label className="text-sm font-medium">Name</label>
              <p className="text-sm">{organization.name}</p>
            </div>
            <div>
              <label className="text-sm font-medium">Created</label>
              <p className="text-sm text-muted-foreground">
                {new Date(organization.metadata.createdAt).toLocaleDateString()}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
