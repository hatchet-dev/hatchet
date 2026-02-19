import { generateTenantSlug } from './generate-tenant-slug';
import { NewOrganizationInputForm } from './new-organization-input-form';
import { cloudApi } from '@/lib/api/api';
import {
  Organization,
  OrganizationTenant,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { AppContextValue, useAppContext } from '@/providers/app-context';
import { useState } from 'react';

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
  refetchTenantMemberships,
  refetchOrganizations,
}: {
  organizationName: string;
  tenantName: string;
} & Pick<
  AppContextValue,
  'refetchTenantMemberships' | 'refetchOrganizations'
>) => {
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

  await Promise.all([refetchTenantMemberships(), refetchOrganizations()]);

  return { organization, tenant };
};

export function NewOrganizationSaverForm({
  defaultOrganizationName,
  defaultTenantName,
  afterSave,
}: NewOrganizationSaverFormProps) {
  const [isSaving, setIsSaving] = useState(false);
  const { isCloudEnabled, refetchTenantMemberships, refetchOrganizations } =
    useAppContext();

  if (!isCloudEnabled) {
    // I feel like there should be an assert here instead
    return <div>Cloud is not enabled</div>;
  }

  const { handleApiError } = useApiError();

  const handleSubmit = (values: {
    organizationName: string;
    tenantName: string;
  }) => {
    setIsSaving(true);

    saveOrganizationAndTenant({
      ...values,
      refetchTenantMemberships,
      refetchOrganizations,
    })
      .then(afterSave)
      .catch(handleApiError)
      .finally(() => setIsSaving(false));
  };

  return (
    <NewOrganizationInputForm
      defaultOrganizationName={defaultOrganizationName}
      defaultTenantName={defaultTenantName}
      isSaving={isSaving}
      onSubmit={handleSubmit}
    />
  );
}
