import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { Loading } from '@/components/v1/ui/loading';
import { Button } from '@/components/v1/ui/button';
import {
  ArrowLeftIcon,
  BuildingOffice2Icon,
} from '@heroicons/react/24/outline';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';

export default function AddTenantPage() {
  const { organization: orgId } = useParams<{ organization: string }>();
  const navigate = useNavigate();
  const [tenantName, setTenantName] = useState('');
  const { handleApiError } = useApiError({});

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

  const createTenantMutation = useMutation({
    mutationFn: async (data: { name: string }) => {
      if (!orgId) {
        throw new Error('Organization ID is required');
      }
      // This would need to be implemented in the API - creating tenant within org
      const result = await cloudApi.tenantCreate({
        name: data.name,
        organizationId: orgId,
      });
      return result.data;
    },
    onSuccess: () => {
      // Navigate back to org page
      navigate(`/organizations/${orgId}`);
    },
    onError: handleApiError,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (tenantName.trim()) {
      createTenantMutation.mutate({ name: tenantName.trim() });
    }
  };

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
    <div className="p-6 max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate(`/organizations/${orgId}`)}
        >
          <ArrowLeftIcon className="h-4 w-4 mr-2" />
          Back
        </Button>
        <div className="flex items-center gap-3">
          <BuildingOffice2Icon className="h-6 w-6 text-primary" />
          <div>
            <h1 className="text-2xl font-bold">Add Tenant</h1>
            <p className="text-muted-foreground">
              Add a new tenant to {organization.name}
            </p>
          </div>
        </div>
      </div>

      {/* Create Tenant Form */}
      <Card>
        <CardHeader>
          <CardTitle>Create New Tenant</CardTitle>
          <CardDescription>
            Enter the details for your new tenant within this organization.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tenantName">Tenant Name</Label>
              <Input
                id="tenantName"
                type="text"
                placeholder="Enter tenant name"
                value={tenantName}
                onChange={(e) => setTenantName(e.target.value)}
                required
              />
              <p className="text-sm text-muted-foreground">
                Choose a descriptive name for your tenant.
              </p>
            </div>

            <div className="space-y-2">
              <Label>Organization</Label>
              <div className="flex items-center gap-2 p-3 bg-muted rounded-md">
                <BuildingOffice2Icon className="h-4 w-4" />
                <span className="font-medium">{organization.name}</span>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => navigate(`/organizations/${orgId}`)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={!tenantName.trim() || createTenantMutation.isPending}
              >
                {createTenantMutation.isPending
                  ? 'Creating...'
                  : 'Create Tenant'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Existing Tenants Info */}
      {organization.tenants && organization.tenants.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Existing Tenants</CardTitle>
            <CardDescription>
              Current tenants in {organization.name}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {organization.tenants.map((tenant) => (
                <div
                  key={tenant.id}
                  className="flex items-center gap-2 p-2 text-sm"
                >
                  <div className="w-2 h-2 rounded-full bg-primary" />
                  <span>{tenant.name}</span>
                  <span className="text-muted-foreground">({tenant.slug})</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
