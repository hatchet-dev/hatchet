import { generateTenantSlug } from './generate-tenant-slug';
import { NewOrganizationInputForm } from './new-organization-input-form';
import {
  OrganizationOnboardingAnswers,
  OrganizationOnboardingQuestionsForm,
  shuffledAttributionOptions,
} from './organization-onboarding-questions-form';
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
import { useMemo, useState } from 'react';
import invariant from 'tiny-invariant';

interface NewOrganizationSaverFormProps {
  defaultOrganizationName?: string;
  defaultTenantName?: string;
  // Whether to ask the "How did you hear about us?" attribution question. Only
  // set on initial (first-org) signup; additional org creation skips it.
  askAttribution?: boolean;
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
      attribution,
      attributionOther,
    }: {
      organizationName: string;
      tenantName: string;
      region?: string;
    } & OrganizationOnboardingAnswers) => {
      const organization = await orgApi
        .organizationCreateMutation()
        .mutationFn({
          name: organizationName,
          ...(isControlPlaneEnabled && attribution?.length
            ? { attribution }
            : {}),
          ...(isControlPlaneEnabled && attributionOther
            ? { attributionOther }
            : {}),
        });
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

type OrganizationDetails = {
  organizationName: string;
  tenantName: string;
  region?: string;
};

export function NewOrganizationSaverForm({
  defaultOrganizationName,
  defaultTenantName,
  askAttribution = false,
  afterSave,
}: NewOrganizationSaverFormProps) {
  const { isLoaded: isUserUniverseLoaded } = useUserUniverse();
  const { isControlPlaneEnabled } = useControlPlane();
  const orgApi = useOrganizationApi();

  const shardsQuery = useQuery({
    ...orgApi.sharedShardsQuery(),
    enabled: isControlPlaneEnabled,
  });

  const saveOrganizationMutation = useSaveOrganization({ afterSave });

  const [details, setDetails] = useState<OrganizationDetails | null>(null);
  const [answers, setAnswers] = useState<OrganizationOnboardingAnswers>({});
  const [step, setStep] = useState<'details' | 'questions'>('details');

  // Shuffle the attribution options once and hold the order stable for this
  // form's lifetime, so it does not reshuffle on re-render or when the user
  // steps back from the questions step and returns.
  const attributionOptions = useMemo(() => shuffledAttributionOptions(), []);

  if (!isUserUniverseLoaded) {
    return <></>;
  }

  invariant(
    isControlPlaneEnabled,
    'NewOrganizationSaverForm requires the control plane',
  );

  const isSaving =
    // Stay in the saving state after success too: the component only
    // unmounts once the post-save navigation commits.
    saveOrganizationMutation.isPending || saveOrganizationMutation.isSuccess;

  if (askAttribution && step === 'questions' && details) {
    return (
      <OrganizationOnboardingQuestionsForm
        isSaving={isSaving}
        options={attributionOptions}
        defaultAnswers={answers}
        onBack={(current) => {
          setAnswers(current);
          setStep('details');
        }}
        onSubmit={(current) =>
          saveOrganizationMutation.mutate({ ...details, ...current })
        }
      />
    );
  }

  return (
    <NewOrganizationInputForm
      defaultOrganizationName={
        details?.organizationName ?? defaultOrganizationName
      }
      defaultTenantName={details?.tenantName ?? defaultTenantName}
      defaultRegion={details?.region}
      isSaving={isSaving}
      // When asking attribution, advance to the questions step; otherwise the
      // input form is the whole flow and submits directly.
      onSubmit={
        askAttribution
          ? (values) => {
              setDetails(values);
              setStep('questions');
            }
          : saveOrganizationMutation.mutate
      }
      showRegionSelect={isControlPlaneEnabled}
      availableShards={shardsQuery.data?.rows}
      isShardsLoading={shardsQuery.isLoading}
      submitLabel={askAttribution ? 'Continue' : 'Get started'}
    />
  );
}
