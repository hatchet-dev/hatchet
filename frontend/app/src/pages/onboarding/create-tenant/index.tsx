import useCloud from '../../auth/hooks/use-cloud';
import { TenantCreateForm } from './components/tenant-create-form';
import { OnboardingFormData } from './types';
import { useOrganizations } from '@/hooks/use-organizations';
import api, {
  CreateTenantRequest,
  queries,
  Tenant,
  TenantEnvironment,
} from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { useSearchParams } from '@/lib/router-helpers';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useState, useEffect } from 'react';

export default function CreateTenant() {
  const [searchParams] = useSearchParams();
  const { organizationData, isCloudEnabled } = useOrganizations();
  const { cloud } = useCloud();

  const organizationId = searchParams.get('organizationId');

  const [selectedOrganizationId, setSelectedOrganizationId] = useState<
    string | null
  >(null);

  // Auto-select organization logic
  useEffect(() => {
    if (organizationData?.rows) {
      const availableOrgs = organizationData.rows.filter((org) => org.isOwner);

      // If organizationId from URL is valid, use it
      if (
        organizationId &&
        availableOrgs.find((org) => org.metadata.id === organizationId)
      ) {
        setSelectedOrganizationId(organizationId);
      }
      // If there's only one organization, auto-select it
      else if (availableOrgs.length === 1) {
        setSelectedOrganizationId(availableOrgs[0].metadata.id);
      }
      // Otherwise, leave it null to show placeholder (ignoring invalid org IDs)
    }
  }, [organizationData, organizationId]);

  const [formData, setFormData] = useState<OnboardingFormData>({
    name: '',
    slug: '',
    environment: TenantEnvironment.Development,
    tenantData: { name: '', environment: TenantEnvironment.Development },
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });
  const navigate = useNavigate();

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const createMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: CreateTenantRequest) => {
      // Use cloud API if cloud is enabled and organization is selected
      if (cloud && selectedOrganizationId) {
        const result = await cloudApi.organizationCreateTenant(
          selectedOrganizationId,
          {
            name: data.name,
            slug: data.slug,
          },
        );

        return { type: 'cloud', data: result.data };
      } else {
        // Use regular API for self-hosted
        const tenant = await api.tenantCreate(data);

        return { type: 'regular', data: tenant.data };
      }
    },
    onSuccess: async (result) => {
      await listMembershipsQuery.refetch();

      setTimeout(() => {
        if (result.type === 'cloud') {
          const tenant = result.data as OrganizationTenant;
          navigate({
            to: appRoutes.tenantOnboardingGetStartedRoute.to,
            params: { tenant: tenant.id },
          });
          return;
        }

        const tenant = result.data as Tenant;
        navigate({
          to: appRoutes.tenantOnboardingGetStartedRoute.to,
          params: { tenant: tenant.metadata.id },
        });
      }, 0);
    },
    onError: handleApiError,
  });

  const validateForm = (): boolean => {
    const { name } = formData.tenantData;
    const errors: Record<string, string> = {};

    // Basic validation for name
    if (!name || name.length < 4 || name.length > 32) {
      errors.name = 'Name must be between 4 and 32 characters';
    }

    if (isCloudEnabled && !selectedOrganizationId) {
      errors.organizationId = 'Please select an organization';
    }

    // Set errors if any exist
    setFieldErrors(errors);

    // Return false if there are any errors
    return Object.keys(errors).length === 0;
  };

  const generateSlug = (name: string): string => {
    return (
      name
        .toLowerCase()
        .replace(/[^a-z0-9-]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '') +
      '-' +
      Math.random().toString(36).substring(0, 5)
    );
  };

  const handleTenantCreate = (tenantData: {
    name: string;
    environment: TenantEnvironment;
  }) => {
    if (!validateForm()) {
      return;
    }

    // Generate slug from name
    const slug = generateSlug(tenantData.name);

    createMutation.mutate({
      name: tenantData.name,
      slug,
      environment: tenantData.environment,
    });
  };

  const updateFormData = (value: { name: string; environment: string }) => {
    setFormData({
      ...formData,
      tenantData: {
        name: value.name,
        environment: value.environment as TenantEnvironment,
      },
      name: value.name,
      environment: value.environment as TenantEnvironment,
    });
  };

  return (
    <div className="flex min-h-full w-full flex-col items-center justify-center p-4">
      <div className="w-full max-w-[450px] space-y-6">
        <div className="flex flex-col space-y-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">
            Create a new tenant
          </h1>
          <p className="text-sm text-muted-foreground">
            A tenant is an isolated environment for your data and workflows.
          </p>
        </div>

        <TenantCreateForm
          value={formData.tenantData}
          onChange={updateFormData}
          onNext={() => handleTenantCreate(formData.tenantData)}
          isLoading={createMutation.isPending}
          fieldErrors={fieldErrors}
          formData={formData}
          setFormData={setFormData}
          className=""
          organizationList={organizationData}
          selectedOrganizationId={selectedOrganizationId}
          onOrganizationChange={setSelectedOrganizationId}
          isCloudEnabled={isCloudEnabled}
        />
      </div>
    </div>
  );
}
