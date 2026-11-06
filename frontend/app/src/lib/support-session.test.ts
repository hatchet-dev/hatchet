import {
  clearSupportSession,
  getSupportSession,
  isSupportSessionExpired,
  markSupportSessionExpired,
  setSupportSession,
  SUPPORT_SESSION_STORAGE_KEY,
  supportRoleLabel,
} from './support-session';
import { SupportGrantRole } from '@/lib/api/generated/control-plane/data-contracts';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const session = {
  token: 'support-token',
  apiUrl: 'https://shard.example.com',
  expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
  grant: {
    id: '44444444-4444-4444-8444-444444444444',
    tenantId: '11111111-1111-4111-8111-111111111111',
    tenantName: 'Tenant A',
    tenantSlug: 'tenant-a',
    role: SupportGrantRole.SUPPORT_OPERATOR,
    reason: 'incident',
    staffEmail: 'staff@example.com',
    createdAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 4 * 60 * 60_000).toISOString(),
  },
};

function withSessionStorage(fn: () => void) {
  const previousSessionStorage = globalThis.sessionStorage;
  const store = new Map<string, string>();

  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => {
        store.set(key, value);
      },
      removeItem: (key: string) => {
        store.delete(key);
      },
    },
  });

  try {
    fn();
  } finally {
    if (previousSessionStorage) {
      Object.defineProperty(globalThis, 'sessionStorage', {
        configurable: true,
        value: previousSessionStorage,
      });
    } else {
      delete (globalThis as Partial<typeof globalThis>).sessionStorage;
    }
  }
}

test('getSupportSession returns null without storage', () => {
  assert.equal(getSupportSession(), null);
  assert.equal(isSupportSessionExpired(), false);
});

test('setSupportSession stores the session per tab and returns a stable snapshot', () => {
  withSessionStorage(() => {
    setSupportSession(session);

    assert.deepEqual(getSupportSession(), session);
    assert.equal(getSupportSession(), getSupportSession());
    assert.equal(isSupportSessionExpired(), false);
    assert.ok(sessionStorage.getItem(SUPPORT_SESSION_STORAGE_KEY));

    clearSupportSession();
    assert.equal(getSupportSession(), null);
  });
});

test('markSupportSessionExpired keeps the session but reports it expired', () => {
  withSessionStorage(() => {
    setSupportSession(session);
    markSupportSessionExpired();

    assert.deepEqual(getSupportSession(), session);
    assert.equal(isSupportSessionExpired(), true);

    clearSupportSession();
  });
});

test('isSupportSessionExpired follows the token expiry', () => {
  withSessionStorage(() => {
    setSupportSession({
      ...session,
      expiresAt: new Date(Date.now() - 1000).toISOString(),
    });

    assert.equal(isSupportSessionExpired(), true);

    clearSupportSession();
  });
});

test('supportRoleLabel maps grant roles to banner labels', () => {
  assert.equal(supportRoleLabel(SupportGrantRole.SUPPORT_VIEWER), 'viewer');
  assert.equal(supportRoleLabel(SupportGrantRole.SUPPORT_OPERATOR), 'operator');
});
