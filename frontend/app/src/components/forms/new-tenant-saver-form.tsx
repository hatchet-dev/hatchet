import { generateTenantSlug } from './generate-tenant-slug';
import { NewTenantInputForm } from './new-tenant-input-form';
import {
  WELCOME_KEY,
  WELCOME_TRIGGER,
} from '@/components/modals/welcome-modal-state';
import { UpgradeRequiredCard } from '@/components/v1/cloud/billing/upgrade-required';
import { useAnalytics } from '@/hooks/use-analytics';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizationEntitlements } from '@/hooks/use-organization-entitlements';
import api, { Tenant } from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useEffect, useState } from 'react';
import invariant from 'tiny-invariant';

type NewTenantSaverFormProps = {
  defaultTenantName?: string;
  defaultOrganizationId?: string;
  afterSave: (
    data:
      | { type: 'cloud'; tenant: OrganizationTenant; organizationId: string }
      | { type: 'regular'; tenant: Tenant },
  ) => void;
  // Called when the user leaves the form to upgrade (e.g. so a host modal can
  // close before navigating to billing).
  onUpgradeNavigate?: () => void;
};

const useSaveTenant = ({
  afterSave,
  onLimitReached,
}: {
  afterSave: NewTenantSaverFormProps['afterSave'];
  onLimitReached?: () => void;
}) => {
  const { isCloudEnabled, invalidate: invalidateUserUniverse } =
    useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();
  const { capture } = useAnalytics();
  const { handleApiError } = useApiError();
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      tenantName,
      organizationId,
      region,
    }: {
      tenantName: string;
      organizationId?: string;
      region?: string;
    }) => {
      const slug = generateTenantSlug(tenantName);
      if (isCloudEnabled) {
        invariant(
          organizationId,
          'Organization ID is required when isCloudEnabled',
        );
        const tenant = await orgApi
          .organizationCreateTenantMutation(organizationId)
          .mutationFn({
            name: tenantName,
            slug,
            ...(isControlPlaneEnabled && region ? { region } : {}),
          });
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
        localStorage.setItem(WELCOME_KEY, WELCOME_TRIGGER.TenantCreated);
        queryClient.invalidateQueries({
          queryKey: ['organization:entitlements:get', data.organizationId],
        });
      }
      capture('onboarding_tenant_created', {
        tenant_type: data.type,
        is_cloud: data.type === 'cloud',
      });
      afterSave(data);
    },
    onError: (error) => {
      if (error instanceof AxiosError && error.response?.status === 403) {
        onLimitReached?.();
        return;
      }
      handleApiError(error as AxiosError);
    },
  });
};

export function NewTenantSaverForm({
  defaultTenantName,
  defaultOrganizationId,
  afterSave,
  onUpgradeNavigate,
}: NewTenantSaverFormProps) {
  const {
    isCloudEnabled,
    organizations,
    isLoaded: isUserUniverseLoaded,
  } = useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();
  const [selectedOrgId, setSelectedOrgId] = useState<string | undefined>(
    defaultOrganizationId,
  );
  const [limitReached, setLimitReached] = useState(false);

  useEffect(() => {
    setSelectedOrgId(defaultOrganizationId);
  }, [defaultOrganizationId]);

  const { canCreateTenant } = useOrganizationEntitlements(selectedOrgId);

  const shardsQuery = useQuery({
    queryKey: ['organization:available-shards', selectedOrgId ?? ''] as const,
    queryFn: async () =>
      (await controlPlaneApi.organizationListAvailableShards(selectedOrgId!))
        .data,
    enabled: Boolean(isCloudEnabled && isControlPlaneEnabled && selectedOrgId),
  });

  const saveTenantMutation = useSaveTenant({
    afterSave,
    onLimitReached: () => setLimitReached(true),
  });

  if (!isUserUniverseLoaded) {
    return <></>;
  }

  invariant(!isCloudEnabled || organizations);

  if (!isCloudEnabled) {
    return (
      <NewTenantInputForm
        defaultTenantName={defaultTenantName}
        isSaving={saveTenantMutation.isPending}
        isCloudEnabled={false}
        onSubmit={saveTenantMutation.mutate}
      />
    );
  }

  if (selectedOrgId && (limitReached || !canCreateTenant)) {
    return (
      <UpgradeRequiredCard
        resource="tenants"
        organizationId={selectedOrgId}
        onNavigate={onUpgradeNavigate}
      />
    );
  }

  return (
    <NewTenantInputForm
      defaultTenantName={defaultTenantName}
      isSaving={saveTenantMutation.isPending}
      isCloudEnabled={true}
      organizations={organizations}
      organizationId={selectedOrgId}
      onOrganizationIdChange={setSelectedOrgId}
      showRegionSelect={isControlPlaneEnabled}
      availableShards={shardsQuery.data?.rows}
      isShardsLoading={shardsQuery.isLoading}
      onSubmit={saveTenantMutation.mutate}
    />
  );
}
