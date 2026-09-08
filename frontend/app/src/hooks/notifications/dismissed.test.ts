import {
  billingUsageDismissKey,
  isLimitNotificationDismissed,
  tenantResourceDismissKey,
} from './dismissed';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('limit notification dismiss keys', () => {
  it('builds stable billing and tenant keys', () => {
    assert.equal(
      billingUsageDismissKey('org-1', 'users', 'exhausted'),
      'billing:org-1:users:exhausted',
    );
    assert.equal(
      tenantResourceDismissKey('tenant-1', 'WORKER', 'warn'),
      'tenant:tenant-1:WORKER:warn',
    );
  });

  it('treats missing and unset keys as visible', () => {
    const dismissed = { 'billing:org-1:users:exhausted': true } as const;

    assert.equal(
      isLimitNotificationDismissed({}, 'billing:org-1:users:exhausted'),
      false,
    );
    assert.equal(isLimitNotificationDismissed(dismissed), false);
    assert.equal(
      isLimitNotificationDismissed(dismissed, 'billing:org-1:users:exhausted'),
      true,
    );
  });
});
