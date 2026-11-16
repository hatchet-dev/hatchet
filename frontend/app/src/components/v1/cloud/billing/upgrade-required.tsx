import { resolveSubscriptionPlanCode } from './subscription-plan-code';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

function nonempty(value: string | null | undefined) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

/** "Free" → "free tier", "Free Plan" → "free plan" */
export function formatPlanTier(planName: string | null | undefined) {
  const lower = nonempty(planName)?.toLowerCase();
  if (!lower) {
    return 'the free tier';
  }
  if (lower.endsWith('tier') || lower.endsWith('plan')) {
    return `the ${lower}`;
  }

  return `the ${lower} tier`;
}

export function useOrganizationBilling(organizationId?: string | null) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();

  return useQuery({
    ...queries.controlPlane.billing(organizationId ?? ''),
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
  });
}

export function useCurrentPlanName(organizationId?: string | null) {
  const billingState = useOrganizationBilling(organizationId);

  const currentPlanCode = nonempty(
    resolveSubscriptionPlanCode(billingState.data?.currentSubscription, null),
  );
  if (currentPlanCode) {
    return (
      billingState.data?.plans?.find((p) => p.planCode === currentPlanCode)
        ?.name ?? currentPlanCode
    );
  }

  const plan = nonempty(billingState.data?.currentSubscription?.plan);
  if (plan) {
    return (
      billingState.data?.plans?.find((p) => p.planCode === plan)?.name ?? plan
    );
  }

  // No subscription row (or an empty currentSubscription) is the free tier.
  return (
    billingState.data?.plans?.find((p) => p.planCode === 'free')?.name ?? 'Free'
  );
}
