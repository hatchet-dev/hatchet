import { generateTenantSlug } from './generate-tenant-slug';
import { NewOrganizationInputForm } from './new-organization-input-form';
import { useAnalytics } from '@/hooks/use-analytics';
import { cloudApi } from '@/lib/api/api';
import {
  Organization,
  OrganizationTenant,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { useMutation } from '@tanstack/react-query';
import invariant from 'tiny-invariant';

interface NewOrganizationSaverFormProps {
  defaultOrganizationName?: string;
  defaultTenantName?: string;
  afterSave: (data: {
    organization: Organization;
    tenant: OrganizationTenant;
  }) => void;
}

const saveOrganizationAndTenant = async ({
  organizationName,
  tenantName,
}: {
  organizationName: string;
  tenantName: string;
}) => {
  const { data: organization } = await cloudApi.organizationCreate({
    name: organizationName,
  });

  const { data: tenant } = await cloudApi.organizationCreateTenant(
    organization.metadata.id,
    {
      name: tenantName,
      slug: generateTenantSlug(tenantName),
    },
  );

  return { organization, tenant };
};

const useSaveOrganization = ({
  afterSave,
}: {
  afterSave: NewOrganizationSaverFormProps['afterSave'];
}) => {
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
  const { capture } = useAnalytics();
  const { handleApiError } = useApiError();

  return useMutation({
    mutationFn: async ({
      organizationName,
      tenantName,
    }: {
      organizationName: string;
      tenantName: string;
    }) =>
      saveOrganizationAndTenant({
        organizationName,
        tenantName,
      }),
    onSuccess: (data) => {
      invalidateUserUniverse();
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
    />
  );
}
