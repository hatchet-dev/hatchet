import { resolveSubscriptionPlanCode } from './subscription-plan-code';
import {
  OrganizationBillingStateSubscription,
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
