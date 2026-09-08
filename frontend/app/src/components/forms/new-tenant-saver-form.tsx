import { generateTenantSlug } from './generate-tenant-slug';
import { NewTenantInputForm } from './new-tenant-input-form';
import {
  WELCOME_KEY,
  WELCOME_TRIGGER,
} from '@/components/modals/welcome-modal-state';
import { UpgradeGateContent } from '@/components/v1/cloud/billing/upgrade-gate-dialog';
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
  allTenantTags?: string[];
  afterSave: (
    data:
      | { type: 'cloud'; tenant: OrganizationTenant; organizationId: string }
      | { type: 'regular'; tenant: Tenant },
  ) => void;
  onUpgradeNavigate?: () => void;
  onGateChange?: (gated: boolean) => void;
};

const useSaveTenant = ({
  afterSave,
  onLimitReached,
}: {
  afterSave: NewTenantSaverFormProps['afterSave'];
  onLimitReached?: () => void;
}) => {
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
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
      tags,
    }: {
      tenantName: string;
      organizationId?: string;
      region?: string;
      tags?: string[];
    }) => {
      const slug = generateTenantSlug(tenantName);
      if (isControlPlaneEnabled) {
        invariant(
          organizationId,
          'Organization ID is required under the control plane',
        );
        const tenant = await orgApi
          .organizationCreateTenantMutation(organizationId)
          .mutationFn({
            name: tenantName,
            slug,
            ...(isControlPlaneEnabled && region ? { region } : {}),
            ...(isControlPlaneEnabled && tags && tags.length > 0
              ? { tags }
              : {}),
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
  allTenantTags,
  afterSave,
  onUpgradeNavigate,
  onGateChange,
}: NewTenantSaverFormProps) {
  const { organizations, isLoaded: isUserUniverseLoaded } = useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();
  const [selectedOrgId, setSelectedOrgId] = useState<string | undefined>(
    defaultOrganizationId,
  );
  const [limitReached, setLimitReached] = useState(false);

  useEffect(() => {
    setSelectedOrgId(defaultOrganizationId);
  }, [defaultOrganizationId]);

  const { canCreateTenant } = useOrganizationEntitlements(selectedOrgId);
  const isGated =
    isControlPlaneEnabled &&
    !!selectedOrgId &&
    (limitReached || !canCreateTenant);

  useEffect(() => {
    onGateChange?.(isGated);
    return () => onGateChange?.(false);
  }, [isGated, onGateChange]);

  const shardsQuery = useQuery({
    queryKey: ['organization:available-shards', selectedOrgId ?? ''] as const,
    queryFn: async () =>
      (await controlPlaneApi.organizationListAvailableShards(selectedOrgId!))
        .data,
    enabled: Boolean(isControlPlaneEnabled && selectedOrgId),
  });

  const saveTenantMutation = useSaveTenant({
    afterSave,
    onLimitReached: () => setLimitReached(true),
  });

  if (!isUserUniverseLoaded) {
    return <></>;
  }

  invariant(!isControlPlaneEnabled || organizations);

  if (!isControlPlaneEnabled) {
    return (
      <NewTenantInputForm
        defaultTenantName={defaultTenantName}
        isSaving={saveTenantMutation.isPending}
        isControlPlaneEnabled={false}
        onSubmit={saveTenantMutation.mutate}
      />
    );
  }

  invariant(organizations);

  if (selectedOrgId && (limitReached || !canCreateTenant)) {
    return (
      <UpgradeGateContent
        gate="tenants"
        organizationId={selectedOrgId}
        onDismiss={onUpgradeNavigate}
      />
    );
  }

  return (
    <NewTenantInputForm
      defaultTenantName={defaultTenantName}
      isSaving={saveTenantMutation.isPending}
      isControlPlaneEnabled={true}
      organizations={organizations}
      organizationId={selectedOrgId}
      onOrganizationIdChange={setSelectedOrgId}
      showRegionSelect={isControlPlaneEnabled}
      showTagsInput={isControlPlaneEnabled}
      allTenantTags={allTenantTags}
      availableShards={shardsQuery.data?.rows}
      isShardsLoading={shardsQuery.isLoading}
      onSubmit={saveTenantMutation.mutate}
    />
  );
}
