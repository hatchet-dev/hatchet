import {
  CONTROL_PLANE_TENANT_STORAGE_KEY,
  getApiErrorStatus,
  readTenantIdFromLocation,
  resolveExchangeTokenTenantId,
} from './api';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const tenantA = '11111111-1111-4111-8111-111111111111';
const tenantB = '22222222-2222-4222-8222-222222222222';
const tenantC = '33333333-3333-4333-8333-333333333333';

function withStoredTenant(tenantId: string, fn: () => void) {
  const previousLocalStorage = globalThis.localStorage;

  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) =>
        key === CONTROL_PLANE_TENANT_STORAGE_KEY
          ? JSON.stringify({ metadata: { id: tenantId } })
          : null,
    },
  });

  try {
    fn();
  } finally {
    if (previousLocalStorage) {
      Object.defineProperty(globalThis, 'localStorage', {
        configurable: true,
        value: previousLocalStorage,
      });
    } else {
      delete (globalThis as Partial<typeof globalThis>).localStorage;
    }
  }
}

function withLocation(pathname: string, fn: () => void) {
  const previousWindow = globalThis.window;

  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      location: { pathname },
    },
  });

  try {
    fn();
  } finally {
    if (previousWindow) {
      Object.defineProperty(globalThis, 'window', {
        configurable: true,
        value: previousWindow,
      });
    } else {
      delete (globalThis as Partial<typeof globalThis>).window;
    }
  }
}

test('readTenantIdFromLocation parses tenant page paths', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    assert.equal(readTenantIdFromLocation(), tenantA);
  });
});

test('readTenantIdFromLocation parses tenant settings paths', () => {
  withLocation(`/tenants/${tenantA}/settings/members`, () => {
    assert.equal(readTenantIdFromLocation(), tenantA);
  });
});

test('readTenantIdFromLocation returns null on organization pages', () => {
  withLocation('/organizations/5b9f0665-ad27-4b5c-bf46-cfad1a280b66', () => {
    assert.equal(readTenantIdFromLocation(), null);
  });
});

test('readTenantIdFromLocation ignores non-UUID tenant segments', () => {
  withLocation('/tenants/not-a-uuid/runs', () => {
    assert.equal(readTenantIdFromLocation(), null);
  });
});

test('resolveExchangeTokenTenantId prefers explicit tenant over location and storage', () => {
  withLocation(`/tenants/${tenantB}/runs`, () => {
    withStoredTenant(tenantC, () => {
      assert.equal(
        resolveExchangeTokenTenantId({
          xTenantId: tenantA,
        }),
        tenantA,
      );
    });
  });
});

test('resolveExchangeTokenTenantId prefers request path tenant over location and storage', () => {
  withLocation(`/tenants/${tenantB}/runs`, () => {
    withStoredTenant(tenantC, () => {
      assert.equal(
        resolveExchangeTokenTenantId({
          url: `/api/v1/stable/tenants/${tenantA}/workflow-runs`,
        }),
        tenantA,
      );
      assert.equal(
        resolveExchangeTokenTenantId({
          url: `/api/v1/tenants/${tenantA}?limit=1`,
        }),
        tenantA,
      );
    });
  });
});

test('resolveExchangeTokenTenantId prefers explicit tenant over request path', () => {
  assert.equal(
    resolveExchangeTokenTenantId({
      xTenantId: tenantA,
      url: `/api/v1/tenants/${tenantB}/workflows`,
    }),
    tenantA,
  );
});

test('resolveExchangeTokenTenantId falls back to location when the request path has no tenant', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    withStoredTenant(tenantB, () => {
      assert.equal(
        resolveExchangeTokenTenantId({
          url: '/api/v1/stable/tasks/44444444-4444-4444-8444-444444444444',
        }),
        tenantA,
      );
      assert.equal(
        resolveExchangeTokenTenantId({
          url: `/api/v1/management/organization-tenants/${tenantC}`,
        }),
        tenantA,
      );
    });
  });
});

test('resolveExchangeTokenTenantId prefers location tenant over storage', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    withStoredTenant(tenantB, () => {
      assert.equal(resolveExchangeTokenTenantId({}), tenantA);
    });
  });
});

test('resolveExchangeTokenTenantId falls back to storage on organization pages', () => {
  withLocation('/organizations/5b9f0665-ad27-4b5c-bf46-cfad1a280b66', () => {
    withStoredTenant(tenantA, () => {
      assert.equal(resolveExchangeTokenTenantId({}), tenantA);
    });
  });
});

test('resolveExchangeTokenTenantId uses location tenant instead of stale storage', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    withStoredTenant(tenantC, () => {
      assert.equal(resolveExchangeTokenTenantId({}), tenantA);
    });
  });
});

test('getApiErrorStatus returns the HTTP response status', () => {
  assert.equal(getApiErrorStatus({ response: { status: 401 } }), 401);
  assert.equal(getApiErrorStatus({ status: 403 }), 403);
});

test('getApiErrorStatus ignores non-HTTP errors', () => {
  assert.equal(getApiErrorStatus(new Error('boom')), undefined);
  assert.equal(getApiErrorStatus({ response: { status: '401' } }), undefined);
});
