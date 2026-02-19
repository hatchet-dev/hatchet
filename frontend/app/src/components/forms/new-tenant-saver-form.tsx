import { generateTenantSlug } from './generate-tenant-slug';
import { NewTenantInputForm } from './new-tenant-input-form';
import { useAnalytics } from '@/hooks/use-analytics';
import api, { Tenant } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import assert from '@/lib/assert';
import { useApiError } from '@/lib/hooks';
import { AppContextValue, useAppContext } from '@/providers/app-context';
import { useState } from 'react';

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
  refetchTenantMemberships,
  refetchOrganizations,
}: {
  tenantName: string;
  organizationId?: string;
  isCloudEnabled: boolean;
} & Pick<
  AppContextValue,
  'refetchTenantMemberships' | 'refetchOrganizations'
>): Promise<FirstArgument<NewTenantSaverFormProps['afterSave']>> => {
  const slug = generateTenantSlug(tenantName);

  if (isCloudEnabled) {
    assert(organizationId, 'Organization ID is required when isCloudEnabled');

    const { data: tenant } = await cloudApi.organizationCreateTenant(
      organizationId,
      {
        name: tenantName,
        slug,
      },
    );

    await Promise.all([refetchTenantMemberships(), refetchOrganizations()]);

    return { type: 'cloud', tenant, organizationId };
  } else {
    const { data: tenant } = await api.tenantCreate({
      name: tenantName,
      slug,
    });

    await refetchTenantMemberships();

    return { type: 'regular', tenant };
  }
};

export function NewTenantSaverForm({
  defaultTenantName,
  defaultOrganizationId,
  afterSave,
}: NewTenantSaverFormProps) {
  const [isSaving, setIsSaving] = useState(false);
  const {
    isCloudEnabled,
    organizations,
    isOrganizationsLoading,
    refetchTenantMemberships,
    refetchOrganizations,
  } = useAppContext();
  const { capture } = useAnalytics();

  if (isOrganizationsLoading) {
    return <></>;
  }

  assert(organizations);

  const { handleApiError } = useApiError();

  const handleSubmit = (values: {
    tenantName: string;
    organizationId?: string;
  }) => {
    setIsSaving(true);

    saveTenant({
      ...values,
      isCloudEnabled,
      refetchTenantMemberships,
      refetchOrganizations,
    })
      .then((result) => {
        capture('onboarding_tenant_created', {
          tenant_type: result.type,
          is_cloud: result.type === 'cloud',
        });
        afterSave(result);
      })
      .catch(handleApiError)
      .finally(() => setIsSaving(false));
  };

  return (
    <NewTenantInputForm
      defaultTenantName={defaultTenantName}
      defaultOrganizationId={defaultOrganizationId}
      isSaving={isSaving}
      isCloudEnabled={isCloudEnabled}
      organizations={organizations.rows}
      onSubmit={handleSubmit}
    />
  );
}
