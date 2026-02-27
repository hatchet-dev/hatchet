import { generateTenantSlug } from './generate-tenant-slug';
import { NewTenantInputForm } from './new-tenant-input-form';
import { useAnalytics } from '@/hooks/use-analytics';
import api, { Tenant } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
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

type FirstArgument<SomeFunction> = SomeFunction extends (
  arg: infer U,
  ...args: any[]
) => any
  ? U
  : never;

const saveTenant = async ({
  tenantName,
  organizationId,
  isCloudEnabled,
}: {
  tenantName: string;
  organizationId?: string;
  isCloudEnabled: boolean;
}): Promise<FirstArgument<NewTenantSaverFormProps['afterSave']>> => {
  const slug = generateTenantSlug(tenantName);

  if (isCloudEnabled) {
    invariant(
      organizationId,
      'Organization ID is required when isCloudEnabled',
    );

    const { data: tenant } = await cloudApi.organizationCreateTenant(
      organizationId,
      {
        name: tenantName,
        slug,
      },
    );

    return { type: 'cloud', tenant, organizationId };
  } else {
    const { data: tenant } = await api.tenantCreate({
      name: tenantName,
      slug,
    });

    return { type: 'regular', tenant };
  }
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

  return useMutation({
    mutationFn: async ({
      tenantName,
      organizationId,
    }: {
      tenantName: string;
      organizationId?: string;
    }) =>
      saveTenant({
        tenantName,
        organizationId,
        isCloudEnabled,
      }),
    onSuccess: (data) => {
      invalidateUserUniverse();
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
