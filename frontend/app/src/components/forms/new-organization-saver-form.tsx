import { generateTenantSlug } from './generate-tenant-slug';
import { NewOrganizationInputForm } from './new-organization-input-form';
import { useAnalytics } from '@/hooks/use-analytics';
import useControlPlane from '@/hooks/use-control-plane';
import {
  Organization,
  OrganizationTenant,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { useMutation, useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';

interface NewOrganizationSaverFormProps {
  defaultOrganizationName?: string;
  defaultTenantName?: string;
  afterSave: (data: {
    organization: Organization;
    tenant: OrganizationTenant;
  }) => void;
}

const useSaveOrganization = ({
  afterSave,
}: {
  afterSave: NewOrganizationSaverFormProps['afterSave'];
}) => {
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();
  const { capture } = useAnalytics();
  const { handleApiError } = useApiError();
  const orgApi = useOrganizationApi();

  return useMutation({
    mutationFn: async ({
      organizationName,
      tenantName,
      region,
    }: {
      organizationName: string;
      tenantName: string;
      region?: string;
    }) => {
      const organization = await orgApi
        .organizationCreateMutation()
        .mutationFn({ name: organizationName });
      const tenant = await orgApi
        .organizationCreateTenantMutation(organization.metadata.id)
        .mutationFn({
          name: tenantName,
          slug: generateTenantSlug(tenantName),
          ...(isControlPlaneEnabled && region ? { region } : {}),
        });
      return { organization, tenant };
    },
    onSuccess: async (data) => {
      await invalidateUserUniverse();
      // Yield a tick so React can flush the universe context update
      // before afterSave navigates away.
      await new Promise((resolve) => setTimeout(resolve, 0));
      localStorage.setItem('hatchet:show-welcome', '1');
      capture('onboarding_tenant_created', {
        tenant_type: 'cloud',
        is_cloud: true,
      });
      afterSave(data);
    },
    onError: handleApiError,
  });
};

export function NewOrganizationSaverForm({
  defaultOrganizationName,
  defaultTenantName,
  afterSave,
}: NewOrganizationSaverFormProps) {
  const { isLoaded: isUserUniverseLoaded, isCloudEnabled } = useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();
  const orgApi = useOrganizationApi();

  const shardsQuery = useQuery({
    ...orgApi.sharedShardsQuery(),
    enabled: isControlPlaneEnabled,
  });

  const saveOrganizationMutation = useSaveOrganization({ afterSave });

  if (!isUserUniverseLoaded) {
    return <></>;
  }

  invariant(
    isCloudEnabled,
    'Organizations only exist in the cloud environment, thus the NewOrganizationSaverForm should never be rendered except in the cloud environment.  If this throws, a UI dev made a mistake.',
  );

  return (
    <NewOrganizationInputForm
      defaultOrganizationName={defaultOrganizationName}
      defaultTenantName={defaultTenantName}
      isSaving={saveOrganizationMutation.isPending}
      onSubmit={saveOrganizationMutation.mutate}
      showRegionSelect={isControlPlaneEnabled}
      availableShards={shardsQuery.data?.rows}
      isShardsLoading={shardsQuery.isLoading}
    />
  );
}
