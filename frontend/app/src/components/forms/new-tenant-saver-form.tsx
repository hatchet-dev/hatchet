import { generateTenantSlug } from './generate-tenant-slug';
import { NewTenantInputForm } from './new-tenant-input-form';
import { useAnalytics } from '@/hooks/use-analytics';
import api, { Tenant } from '@/lib/api';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { useMutation } from '@tanstack/react-query';
import invariant from 'tiny-invariant';

type NewTenantSaverFormProps = {
  defaultTenantName?: string;
  defaultOrganizationId?: string;
  afterSave: (
    data:
      | { type: 'cloud'; tenant: OrganizationTenant; organizationId: string }
      | { type: 'regular'; tenant: Tenant },
  ) => void;
};

const useSaveTenant = ({
  afterSave,
}: {
  afterSave: NewTenantSaverFormProps['afterSave'];
}) => {
  const { isCloudEnabled, invalidate: invalidateUserUniverse } =
    useUserUniverse();
  const { capture } = useAnalytics();
  const { handleApiError } = useApiError();
  const orgApi = useOrganizationApi();

  return useMutation({
    mutationFn: async ({
      tenantName,
      organizationId,
    }: {
      tenantName: string;
      organizationId?: string;
    }) => {
      const slug = generateTenantSlug(tenantName);
      if (isCloudEnabled) {
        invariant(
          organizationId,
          'Organization ID is required when isCloudEnabled',
        );
        const tenant = await orgApi
          .organizationCreateTenantMutation(organizationId)
          .mutationFn({ name: tenantName, slug });
        return { type: 'cloud' as const, tenant, organizationId };
      } else {
        const { data: tenant } = await api.tenantCreate({
          name: tenantName,
          slug,
        });
        return { type: 'regular' as const, tenant };
      }
    },
    onSuccess: async (data) => {
      await invalidateUserUniverse();
      // Yield a tick so React can flush the universe context update
      // before afterSave navigates away.
      await new Promise((resolve) => setTimeout(resolve, 0));
      if (data.type === 'cloud') {
        localStorage.setItem('hatchet:show-welcome', '1');
      }
      capture('onboarding_tenant_created', {
        tenant_type: data.type,
        is_cloud: data.type === 'cloud',
      });
      afterSave(data);
    },
    onError: handleApiError,
  });
};

export function NewTenantSaverForm({
  defaultTenantName,
  defaultOrganizationId,
  afterSave,
}: NewTenantSaverFormProps) {
  const {
    isCloudEnabled,
    organizations,
    isLoaded: isUserUniverseLoaded,
  } = useUserUniverse();

  const saveTenantMutation = useSaveTenant({ afterSave });

  if (!isUserUniverseLoaded) {
    return <></>;
  }

  const props = isCloudEnabled
    ? ({
        isCloudEnabled: true,
        organizations,
      } as const)
    : ({
        isCloudEnabled: false,
      } as const);

  invariant(!isCloudEnabled || organizations);

  return (
    <NewTenantInputForm
      defaultTenantName={defaultTenantName}
      defaultOrganizationId={defaultOrganizationId}
      isSaving={saveTenantMutation.isPending}
      onSubmit={saveTenantMutation.mutate}
      {...props}
    />
  );
}
