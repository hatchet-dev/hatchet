import { canManageApiTokens } from './api-token-permissions';
import { TenantMemberRole } from './api/generated/data-contracts';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('owner and admin can create and revoke API tokens', () => {
  assert.equal(canManageApiTokens(TenantMemberRole.OWNER), true);
  assert.equal(canManageApiTokens(TenantMemberRole.ADMIN), true);
});

test('member sees the token list without create or revoke actions', () => {
  assert.equal(canManageApiTokens(TenantMemberRole.MEMBER), false);
});

test('viewer and unknown membership cannot manage API tokens', () => {
  assert.equal(canManageApiTokens(TenantMemberRole.VIEWER), false);
  assert.equal(canManageApiTokens(undefined), false);
});
