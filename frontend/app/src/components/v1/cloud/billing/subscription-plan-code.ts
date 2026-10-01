import {
  OrganizationBillingStateSubscription,
  SubscriptionPlan,
  SubscriptionPlanCode,
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

function isBasePlan(
  planCode: string | null | undefined,
  code: SubscriptionPlanCode,
) {
  return planCodeBase(planCode) === (code as string);
}

export function isPayAsYouGoPlanCode(planCode?: string | null) {
  return isBasePlan(planCode, SubscriptionPlanCode.PayAsYouGo);
}

export function canSelfServePayAsYouGoUpgrade(planCode?: string | null) {
  return (
    isBasePlan(planCode, SubscriptionPlanCode.Free) ||
    isBasePlan(planCode, SubscriptionPlanCode.Developer)
  );
}

export function payAsYouGoPlan(plans?: SubscriptionPlan[]) {
  return plans?.find((plan) => isPayAsYouGoPlanCode(plan.planCode));
}
