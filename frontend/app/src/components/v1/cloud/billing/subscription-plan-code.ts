import { OrganizationBillingStateSubscription } from '@/lib/api/generated/control-plane/data-contracts';

export function resolveSubscriptionPlanCode(
  subscription: OrganizationBillingStateSubscription | undefined,
  fallback: string | null,
) {
  return subscription?.planCode ?? fallback;
}
