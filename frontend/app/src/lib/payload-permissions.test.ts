import { TenantMemberRole } from './api/generated/data-contracts';
import {
  canViewPayloadsForMember,
  payloadsLockedForRole,
} from './payload-permissions';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('owner and admin roles always see payloads', () => {
  assert.equal(payloadsLockedForRole(TenantMemberRole.OWNER), true);
  assert.equal(payloadsLockedForRole(TenantMemberRole.ADMIN), true);
  assert.equal(payloadsLockedForRole(TenantMemberRole.MEMBER), false);
  assert.equal(payloadsLockedForRole(TenantMemberRole.VIEWER), false);
  assert.equal(payloadsLockedForRole(undefined), false);
});

test('owner and admin ignore a false canViewPayloads flag', () => {
  for (const role of [TenantMemberRole.OWNER, TenantMemberRole.ADMIN]) {
    assert.equal(
      canViewPayloadsForMember({ role, canViewPayloads: false }),
      true,
    );
  }
});

test('other roles follow the canViewPayloads flag', () => {
  for (const role of [TenantMemberRole.MEMBER, TenantMemberRole.VIEWER]) {
    assert.equal(
      canViewPayloadsForMember({ role, canViewPayloads: true }),
      true,
    );
    assert.equal(
      canViewPayloadsForMember({ role, canViewPayloads: false }),
      false,
    );
  }
});

test('an unset flag defaults to allowed', () => {
  assert.equal(
    canViewPayloadsForMember({ role: TenantMemberRole.MEMBER }),
    true,
  );
});

test('an unknown membership defaults to allowed', () => {
  assert.equal(canViewPayloadsForMember(undefined), true);
});
