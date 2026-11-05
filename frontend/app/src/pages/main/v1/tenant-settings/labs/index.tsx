import { LabsFlag, LabsPreview } from './labs-preview';
import { payAsYouGoPlan } from '@/components/v1/cloud/billing/subscription-plan-code';
import useControlPlane from '@/hooks/use-control-plane';
import {
  getPlanChangeErrorMessage,
  useSubscriptionUpgrade,
} from '@/hooks/use-subscription-upgrade';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { useTenantEntitlements } from '@/hooks/use-tenant-entitlements';
import api, {
  queries,
  TenantEntitlements,
  TenantMemberRole,
  UpdateTenantEntitlementsRequest,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { useState } from 'react';

export default function LabsSettings() {
  const { tenantId } = useCurrentTenantId();
  const { membership, billing, organizationId, isUserUniverseLoaded } =
    useTenantDetails();
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const {
    entitlements,
    isLoading,
    error: entitlementsError,
  } = useTenantEntitlements();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [errorMessage, setErrorMessage] = useState<string>();
  const { handleApiError } = useApiError({
    setErrors: (errors) => setErrorMessage(errors.join(' ')),
  });

  const billingPending = canBill && (!billing || billing.isLoading);
  const onFreePlan =
    canBill &&
    !!organizationId &&
    !!billing?.state &&
    !billing.isLoading &&
    billing.plan === 'free';

  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: onFreePlan && isControlPlaneEnabled && canBill,
  });
  const payg = payAsYouGoPlan(plansQuery.data?.plans);
  const upgrade = useSubscriptionUpgrade(organizationId);

  const queryKey = queries.tenantEntitlements.get(tenantId).queryKey;
  const update = useMutation({
    mutationFn: async (patch: UpdateTenantEntitlementsRequest) =>
      (await api.tenantEntitlementsUpdate(tenantId, patch)).data,
    onMutate: async (patch) => {
      setErrorMessage(undefined);
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<TenantEntitlements>(queryKey);
      queryClient.setQueryData<TenantEntitlements>(queryKey, (current) =>
        current ? { ...current, ...patch } : current,
      );
      return { previous };
    },
    onError: (error, _patch, context) => {
      if (context?.previous) {
        queryClient.setQueryData(queryKey, context.previous);
      }
      if (error instanceof AxiosError) {
        handleApiError(error);
        return;
      }
      setErrorMessage('Failed to update Labs settings.');
    },
    onSuccess: (data) => {
      queryClient.setQueryData(queryKey, data);
    },
  });

  if (!isUserUniverseLoaded || isLoading || billingPending) {
    return <LabsPreview status="loading" />;
  }

  if (onFreePlan && organizationId) {
    const upgradeError = upgrade.isError
      ? getPlanChangeErrorMessage(upgrade.error)
      : plansQuery.isError
        ? 'Failed to load plans.'
        : undefined;

    return (
      <LabsPreview
        status="upgrade"
        canUpgrade={!!payg}
        upgradePending={upgrade.isPending}
        error={upgradeError}
        onUpgrade={() => payg && upgrade.mutate(payg.planCode)}
        onViewBilling={() =>
          navigate({
            to: appRoutes.organizationBillingRoute.to,
            params: { organization: organizationId },
          })
        }
      />
    );
  }

  if (entitlementsError || !entitlements) {
    return (
      <LabsPreview status="error" message="Failed to load Labs settings." />
    );
  }

  if (membership !== TenantMemberRole.OWNER) {
    return <LabsPreview status="gated" />;
  }

  return (
    <LabsPreview
      status="ready"
      pending={update.isPending}
      error={errorMessage}
      flags={{
        durableStreams: entitlements.durableStreams,
        sandboxOperator: entitlements.sandboxOperator,
        serverlessOperator: entitlements.serverlessOperator,
      }}
      onToggle={(flag: LabsFlag, enabled: boolean) =>
        update.mutate({ [flag]: enabled })
      }
    />
  );
}
