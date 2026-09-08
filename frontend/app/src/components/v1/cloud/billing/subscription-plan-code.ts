import {
  OrganizationBillingStateSubscription,
  SubscriptionPlan,
} from '@/lib/api/generated/control-plane/data-contracts';

export function resolveSubscriptionPlanCode(
  subscription: OrganizationBillingStateSubscription | undefined,
  fallback: string | null,
) {
  return subscription?.planCode ?? fallback;
}

export function planCodeBase(planCode?: string | null) {
  return (planCode ?? '').split('_')[0];
}

export function isPayAsYouGoPlanCode(planCode?: string | null) {
  return planCodeBase(planCode) === 'pay-as-you-go';
}

export function payAsYouGoPlan(plans?: SubscriptionPlan[]) {
  return plans?.find((plan) => isPayAsYouGoPlanCode(plan.planCode));
}
