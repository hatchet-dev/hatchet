import {
  CONTROL_PLANE_TENANT_STORAGE_KEY,
  readTenantIdFromRequestUrl,
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

test('readTenantIdFromRequestUrl parses tenant API paths', () => {
  assert.equal(
    readTenantIdFromRequestUrl({
      url: `/api/v1/tenants/${tenantA}/workflows`,
    }),
    tenantA,
  );
});

test('readTenantIdFromRequestUrl parses relative paths with empty baseURL', () => {
  assert.equal(
    readTenantIdFromRequestUrl({
      baseURL: '',
      url: `/api/v1/tenants/${tenantA}/workflows`,
    }),
    tenantA,
  );
});

test('readTenantIdFromRequestUrl parses stable tenant API paths', () => {
  assert.equal(
    readTenantIdFromRequestUrl({
      url: `/api/v1/stable/tenants/${tenantA}/task-metrics`,
    }),
    tenantA,
  );
});

test('readTenantIdFromRequestUrl parses absolute URLs', () => {
  assert.equal(
    readTenantIdFromRequestUrl({
      url: `https://cloud.onhatchet.run/api/v1/tenants/${tenantA}/workflows`,
    }),
    tenantA,
  );
});

test('readTenantIdFromRequestUrl ignores non-UUID tenant segments', () => {
  assert.equal(
    readTenantIdFromRequestUrl({
      url: '/api/v1/tenants/not-a-uuid/workflows',
    }),
    null,
  );
});

test('resolveExchangeTokenTenantId prefers explicit tenant over URL and storage', () => {
  withStoredTenant(tenantC, () => {
    assert.equal(
      resolveExchangeTokenTenantId({
        url: `/api/v1/tenants/${tenantB}/workflows`,
        xTenantId: tenantA,
      }),
      tenantA,
    );
  });
});

test('resolveExchangeTokenTenantId prefers URL tenant over storage', () => {
  withStoredTenant(tenantB, () => {
    assert.equal(
      resolveExchangeTokenTenantId({
        url: `/api/v1/tenants/${tenantA}/workflows`,
      }),
      tenantA,
    );
  });
});

test('resolveExchangeTokenTenantId falls back to storage', () => {
  withStoredTenant(tenantA, () => {
    assert.equal(
      resolveExchangeTokenTenantId({
        url: '/api/v1/workflows/workflow-id',
      }),
      tenantA,
    );
  });
});
