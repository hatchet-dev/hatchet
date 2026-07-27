import { generateTenantSlug } from './generate-tenant-slug';
import { NewOrganizationInputForm } from './new-organization-input-form';
import {
  WELCOME_KEY,
  WELCOME_TRIGGER,
} from '@/components/modals/welcome-modal-state';
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
  // May return the navigation promise so the mutation stays pending (and the
  // form stays in its saving state) until the destination page commits.
  afterSave: (data: {
    organization: Organization;
    tenant: OrganizationTenant;
  }) => void | Promise<void>;
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
      localStorage.setItem(WELCOME_KEY, WELCOME_TRIGGER.OrganizationCreated);
      capture('onboarding_tenant_created', {
        tenant_type: 'cloud',
        is_cloud: true,
      });
      // Keep the mutation pending until any navigation in afterSave commits;
      // otherwise the form flashes back to its idle state while the target
      // route's loader is still running.
      await afterSave(data);
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
      isSaving={
        // Stay in the saving state after success too: the component only
        // unmounts once the post-save navigation commits.
        saveOrganizationMutation.isPending || saveOrganizationMutation.isSuccess
      }
      onSubmit={saveOrganizationMutation.mutate}
      showRegionSelect={isControlPlaneEnabled}
      availableShards={shardsQuery.data?.rows}
      isShardsLoading={shardsQuery.isLoading}
    />
  );
}
