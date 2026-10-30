import {
  canSelfServePayAsYouGoUpgrade,
  isPayAsYouGoPlanCode,
  payAsYouGoPlan,
  resolveSubscriptionPlanCode,
} from './subscription-plan-code';
import {
  OrganizationBillingStateSubscription,
  SubscriptionPlan,
  SubscriptionPlanCode,
} from '@/lib/api/generated/control-plane/data-contracts';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

const startedAt = '2026-06-02T13:04:22.935Z';

function subscription(
  plan: OrganizationBillingStateSubscription['plan'],
  planCode: OrganizationBillingStateSubscription['planCode'],
): OrganizationBillingStateSubscription {
  return { plan, planCode, startedAt };
}

describe('resolveSubscriptionPlanCode', () => {
  it('uses the server-resolved subscription plan code', () => {
    assert.equal(
      resolveSubscriptionPlanCode(
        subscription(SubscriptionPlanCode.Team, 'team_monthly'),
        'free',
      ),
      'team_monthly',
    );
  });

  it('supports plan codes that do not include a period', () => {
    assert.equal(
      resolveSubscriptionPlanCode(
        subscription(SubscriptionPlanCode.Developer, 'developer'),
        'free',
      ),
      'developer',
    );
  });

  it('uses the fallback when there is no subscription plan', () => {
    assert.equal(resolveSubscriptionPlanCode(undefined, 'free'), 'free');
    assert.equal(resolveSubscriptionPlanCode(undefined, null), null);
  });
});

describe('isPayAsYouGoPlanCode', () => {
  it('matches the pay-as-you-go plan code', () => {
    assert.equal(isPayAsYouGoPlanCode('pay-as-you-go'), true);
    assert.equal(isPayAsYouGoPlanCode('pay-as-you-go_monthly'), true);
    assert.equal(isPayAsYouGoPlanCode('developer'), false);
  });
});

describe('canSelfServePayAsYouGoUpgrade', () => {
  it('allows free and developer to self-serve to pay-as-you-go', () => {
    assert.equal(canSelfServePayAsYouGoUpgrade('free'), true);
    assert.equal(canSelfServePayAsYouGoUpgrade('developer'), true);
  });

  it('keeps paid legacy plans sales-gated', () => {
    assert.equal(canSelfServePayAsYouGoUpgrade('starter_monthly'), false);
    assert.equal(canSelfServePayAsYouGoUpgrade('growth_yearly'), false);
    assert.equal(canSelfServePayAsYouGoUpgrade('team_monthly'), false);
    assert.equal(canSelfServePayAsYouGoUpgrade('scale_monthly'), false);
    assert.equal(canSelfServePayAsYouGoUpgrade('migration'), false);
    assert.equal(canSelfServePayAsYouGoUpgrade('pay-as-you-go'), false);
  });
});

describe('payAsYouGoPlan', () => {
  it('finds the pay-as-you-go plan in a catalog', () => {
    const plans = [
      { planCode: 'free', name: 'Free', description: '', amountCents: 0 },
      {
        planCode: 'pay-as-you-go',
        name: 'Pay as you Go',
        description: '',
        amountCents: 0,
      },
    ] as SubscriptionPlan[];

    assert.equal(payAsYouGoPlan(plans)?.name, 'Pay as you Go');
  });
});
